//go:build integration

package transfer

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestCompleteAgreement(t *testing.T) {
	db, dsn := completionDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	service := NewService(NewPostgresRepository(db))
	cmd := CompletionCommand{AgreementID: "agreement-1", ActorID: "author", Summary: "Skill exchange complete", Rating: 5}

	for _, tc := range []struct {
		name string
		edit func(*CompletionCommand)
		want error
	}{
		{"invalid", func(c *CompletionCommand) { c.Summary = " " }, ErrInvalidRequest},
		{"forbidden", func(c *CompletionCommand) { c.ActorID = "outsider" }, ErrForbidden},
		{"missing", func(c *CompletionCommand) { c.AgreementID = "missing" }, ErrNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := cmd
			tc.edit(&input)
			if _, err := service.Complete(ctx, input); !errors.Is(err, tc.want) {
				t.Fatalf("error = %v, want %v", err, tc.want)
			}
		})
	}

	first, err := service.Complete(ctx, cmd)
	if err != nil || !first.Changed || first.Status != "completed" || first.Revision != 2 ||
		first.Completion == nil || first.Completion.Summary != cmd.Summary || first.Completion.Rating != cmd.Rating {
		t.Fatalf("complete = %+v, %v", first, err)
	}
	assertReservations(t, db, 0)
	assertOtherAgreement(t, db)

	// Use a new pool and service to verify persistence without process memory.
	reopened, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	service = NewService(NewPostgresRepository(reopened))
	got, err := service.GetAgreement(ctx, cmd.AgreementID, "learner")
	if err != nil || got.Status != "completed" || got.Revision != 2 ||
		got.Completion == nil || *got.Completion != *first.Completion ||
		got.InitiatorSlotID != "" || got.PartnerSlotID != "" {
		t.Fatalf("GET = %+v, %v", got, err)
	}
	retry := cmd
	retry.Summary = "Other text"
	retry.ActorID = "learner"
	again, err := service.Complete(ctx, retry)
	if err != nil || again.Changed || again.Revision != 2 ||
		again.Completion == nil || *again.Completion != *first.Completion {
		t.Fatalf("repeat = %+v, %v", again, err)
	}
	retry.ActorID = "outsider"
	if _, err = service.Complete(ctx, retry); !errors.Is(err, ErrForbidden) {
		t.Fatalf("forbidden repeat: %v", err)
	}
}

func completionDB(t *testing.T) (*sql.DB, string) {
	t.Helper()
	dsn := os.Getenv("SKILLSWAP_TEST_DATABASE_URL")
	if dsn == "" {
		t.Fatal("set SKILLSWAP_TEST_DATABASE_URL to a PostgreSQL test database")
	}
	admin, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { admin.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var schema string
	if err := admin.QueryRowContext(ctx, "SELECT 'completion_' || replace(gen_random_uuid()::text, '-', '')").Scan(&schema); err != nil {
		t.Fatal(err)
	}
	if _, err := admin.ExecContext(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := admin.ExecContext(ctx, "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			t.Error(err)
		}
	})
	separator := "?"
	if strings.Contains(dsn, "?") {
		separator = "&"
	}
	dsn += separator + "search_path=" + schema
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	for _, name := range []string{"001_agreements.sql", "002_completion.sql", "fixtures/demo.sql"} {
		query, err := os.ReadFile(filepath.Join("..", "..", "migrations", name))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.ExecContext(ctx, string(query)); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	return db, dsn
}

func assertReservations(t *testing.T, db *sql.DB, want int) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var count int
	if err := db.QueryRowContext(ctx, "SELECT count(*) FROM reservations WHERE agreement_id='agreement-1'").Scan(&count); err != nil || count != want {
		t.Fatalf("reservations = %d, %v; want %d", count, err, want)
	}
}

func assertOtherAgreement(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	result, err := NewPostgresRepository(db).GetAgreement(ctx, "agreement-2", "author")
	if err != nil || result.Status != "active" || result.Revision != 1 ||
		result.InitiatorSlotID != "slot-5" || result.PartnerSlotID != "slot-6" {
		t.Fatalf("unrelated agreement = %+v, %v", result, err)
	}
}

func TestCompletionRollback(t *testing.T) {
	db, dsn := completionDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// The trigger fails the status update only after both deletes are visible to the writer.
	execSQL(t, db, `
CREATE FUNCTION fail_completion() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM reservations WHERE agreement_id = NEW.id) THEN
        RAISE EXCEPTION 'injected after releasing reservations';
    END IF;
    RETURN NEW;
END $$;
CREATE TRIGGER fail_completion BEFORE UPDATE ON agreements
FOR EACH ROW WHEN (NEW.status = 'completed') EXECUTE FUNCTION fail_completion();`)
	service := NewService(NewPostgresRepository(db))
	_, err := service.Complete(ctx, demoCompletion())
	var pg *pgconn.PgError
	if !errors.As(err, &pg) || pg.Code != "P0001" || pg.Message != "injected after releasing reservations" {
		t.Fatalf("expected failure after reservation deletion, got %v", err)
	}
	observer, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer observer.Close()
	result, err := NewService(NewPostgresRepository(observer)).GetAgreement(ctx, "agreement-1", "author")
	if err != nil || result.Status != "active" || result.Revision != 1 || result.Completion != nil ||
		result.InitiatorSlotID != "slot-1" || result.PartnerSlotID != "slot-2" {
		t.Fatalf("state after rollback = %+v, %v", result, err)
	}
	execSQL(t, db, "DROP TRIGGER fail_completion ON agreements")
	if _, err = service.Complete(ctx, demoCompletion()); err != nil {
		t.Fatalf("completion after rollback: %v", err)
	}
}

func TestCompletionCompetition(t *testing.T) {
	for _, tc := range []struct {
		first, second string
		secondErr     error
		status        string
		revision      int64
		transfers     int
	}{
		{"complete", "transfer", ErrInvalidTransition, "completed", 2, 0},
		{"transfer", "complete", nil, "completed", 3, 1},
		{"complete", "cancel", ErrInvalidTransition, "completed", 2, 0},
		{"cancel", "complete", ErrInvalidTransition, "canceled", 2, 0},
	} {
		t.Run(tc.first+"_then_"+tc.second, func(t *testing.T) {
			db, _ := completionDB(t)
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			defer cancel()
			service := NewService(NewPostgresRepository(db))
			execSQL(t, db, `
CREATE FUNCTION block_update() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    PERFORM pg_advisory_xact_lock(4074);
    RETURN NEW;
END $$;
CREATE TRIGGER block_update BEFORE UPDATE ON agreements
FOR EACH ROW EXECUTE FUNCTION block_update();`)
			gate, err := db.BeginTx(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer gate.Rollback()
			var gatePID int
			if err := gate.QueryRowContext(ctx, "SELECT pg_backend_pid()").Scan(&gatePID); err != nil {
				t.Fatal(err)
			}
			if _, err := gate.ExecContext(ctx, "SELECT pg_advisory_xact_lock(4074)"); err != nil {
				t.Fatal(err)
			}

			run := func(name string) <-chan error {
				done := make(chan error, 1)
				go func() {
					defer close(done)
					var err error
					switch name {
					case "complete":
						_, err = service.Complete(ctx, demoCompletion())
					case "transfer":
						_, err = service.Transfer(ctx, Command{
							AgreementID: "agreement-1", ActorID: "author", CommandID: "transfer-1",
							ProposalID: "proposal-1", ExpectedRevision: 1,
						})
					case "cancel":
						err = cancelFixtureAgreement(ctx, db)
					}
					done <- err
				}()
				t.Cleanup(func() { cancel(); <-done })
				return done
			}
			first := run(tc.first)
			firstPID := waitForBlock(t, ctx, db, gatePID)
			second := run(tc.second)
			waitForBlock(t, ctx, db, firstPID)
			// GET sees the last committed snapshot while the winning writer is blocked.
			before, err := service.GetAgreement(ctx, "agreement-1", "author")
			if err != nil || before.Status != "active" || before.InitiatorSlotID != "slot-1" || before.PartnerSlotID != "slot-2" {
				t.Fatalf("GET during transaction = %+v, %v", before, err)
			}
			if err := gate.Commit(); err != nil {
				t.Fatal(err)
			}
			if err := <-first; err != nil {
				t.Fatalf("first operation: %v", err)
			}
			if err := <-second; !errors.Is(err, tc.secondErr) {
				t.Fatalf("second operation = %v, want %v", err, tc.secondErr)
			}
			result, err := service.GetAgreement(ctx, "agreement-1", "author")
			if err != nil || result.Status != tc.status || result.Revision != tc.revision ||
				(result.Completion != nil) != (tc.status == "completed") {
				t.Fatalf("final agreement = %+v, %v", result, err)
			}
			assertReservations(t, db, 0)
			assertOtherAgreement(t, db)
			var transfers int
			if err := db.QueryRowContext(ctx, "SELECT count(*) FROM transfer_commands").Scan(&transfers); err != nil || transfers != tc.transfers {
				t.Fatalf("saved transfers = %d, %v; want %d", transfers, err, tc.transfers)
			}
		})
	}
}

func TestCompletionLockTimeout(t *testing.T) {
	db, _ := completionDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	gate, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer gate.Rollback()
	// Hold the agreement so the command runs out of its transaction budget.
	if _, err := gate.ExecContext(ctx, "SELECT id FROM agreements WHERE id='agreement-1' FOR UPDATE"); err != nil {
		t.Fatal(err)
	}
	service := NewService(NewPostgresRepository(db))
	short, stop := context.WithTimeout(ctx, 200*time.Millisecond)
	defer stop()
	if _, err := service.Complete(short, demoCompletion()); !errors.Is(err, ErrRetryable) {
		t.Fatalf("lock timeout = %v", err)
	}
	assertReservations(t, db, 2)
	if err := gate.Rollback(); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Complete(ctx, demoCompletion()); err != nil {
		t.Fatalf("completion after timeout: %v", err)
	}
}

func TestCompletionSchema(t *testing.T) {
	db, _ := completionDB(t)
	for _, tc := range []struct{ query, state string }{
		{"UPDATE agreements SET status='completed' WHERE id='agreement-1'", "23514"},
		{"UPDATE agreements SET status='completed', result_summary='done', result_rating=6 WHERE id='agreement-1'", "23514"},
		{"INSERT INTO reservations VALUES ('slot-3','agreement-1','author')", "23505"},
		{"INSERT INTO reservations VALUES ('slot-3','agreement-1','outsider')", "23503"},
	} {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		_, err := db.ExecContext(ctx, tc.query)
		cancel()
		var pg *pgconn.PgError
		if !errors.As(err, &pg) || pg.Code != tc.state {
			t.Fatalf("%s: got %v, want SQLSTATE %s", tc.query, err, tc.state)
		}
	}
}

func demoCompletion() CompletionCommand {
	return CompletionCommand{AgreementID: "agreement-1", ActorID: "author", Summary: "Exchange complete", Rating: 5}
}

func execSQL(t *testing.T, db *sql.DB, query string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := db.ExecContext(ctx, query); err != nil {
		t.Fatal(err)
	}
}

func waitForBlock(t *testing.T, ctx context.Context, db *sql.DB, blocker int) int {
	t.Helper()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		var pid int
		err := db.QueryRowContext(ctx, `
SELECT pid FROM pg_stat_activity
WHERE datname = current_database() AND $1 = ANY(pg_blocking_pids(pid)) LIMIT 1`, blocker).Scan(&pid)
		if err == nil {
			return pid
		}
		if !errors.Is(err, sql.ErrNoRows) {
			t.Fatal(err)
		}
		select {
		case <-ctx.Done():
			t.Fatalf("no backend blocked by %d: %v", blocker, ctx.Err())
		case <-ticker.C:
		}
	}
}

// The team's cancellation endpoint is developed separately. This test writer
// exercises the agreed lock order and terminal state on a separate connection.
func cancelFixtureAgreement(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	a, err := lockAgreement(ctx, tx, "agreement-1", "learner")
	if err != nil {
		return err
	}
	if a.Status != "active" {
		return ErrInvalidTransition
	}
	slots, err := currentReservations(ctx, tx, a)
	if err != nil {
		return err
	}
	if err := lockSlots(ctx, tx, slots[a.InitiatorID], slots[a.PartnerID]); err != nil {
		return err
	}
	if err := releaseReservations(ctx, tx, a.ID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "UPDATE agreements SET status='canceled', revision=revision+1 WHERE id=$1", a.ID); err != nil {
		return err
	}
	return tx.Commit()
}
