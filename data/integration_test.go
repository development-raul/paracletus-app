//go:build integration

// run tests with this command: go test . --tags integration --count=1
package data

import (
	"database/sql"
	"fmt"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgconn"
	_ "github.com/jackc/pgx/v4"
	_ "github.com/jackc/pgx/v4/stdlib"
)

var (
	host     = "localhost"
	user     = "postgres"
	password = "secret"
	dbName   = "paracletus_test"
	port     = "5435"
	dsn      = "host=%s port=%s user=%s password=%s dbname=%s sslmode=disable timezone=UTC connect_timeout=5"
)

var dummyUser = User{
	FirstName: "Some",
	LastName:  "Guy",
	Email:     "me@here.com",
	Active:    1,
	Password:  "password",
}

var models Models
var testDB *sql.DB
var resource *dockertest.Resource
var pool *dockertest.Pool

func TestMain(m *testing.M) {
	os.Setenv("DATABASE_TYPE", "postgres")
	os.Setenv("UPPER_DB_LOG", "ERROR")

	// Get a docker image and run it
	p, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("could not connect to docker: %s", err)
	}
	// Set the package level variable pool as the test pool
	pool = p

	// Set the docker options
	opts := dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "13.4",
		Env: []string{
			"POSTGRES_USER=" + user,
			"POSTGRES_PASSWORD=" + password,
			"POSTGRES_DB=" + dbName,
		},
		ExposedPorts: []string{"5432"},
		PortBindings: map[docker.Port][]docker.PortBinding{
			"5432": {
				{HostIP: "0.0.0.0", HostPort: port},
			},
		},
	}
	// Run docker with the options above
	resource, err := pool.RunWithOptions(&opts)
	if err != nil {
		// Remove the docker image so that it does not hang around
		_ = pool.Purge(resource)
		log.Fatalf("could not start resource: %s", err)
	}

	// Make sure we wait until the docker and db connection are ready
	if err := pool.Retry(func() error {
		var err error
		testDB, err = sql.Open("pgx", fmt.Sprintf(dsn, host, port, user, password, dbName))
		if err != nil {
			return err
		}
		return testDB.Ping()
	}); err != nil {
		_ = pool.Purge(resource)
		log.Fatalf("could not connect to docker: %s", err)
	}

	// We can safely assume that docker and postgres have started
	err = createTables(testDB)
	if err != nil {
		log.Fatalf("error creating tables: %s", err)
	}

	models = New(testDB)

	code := m.Run()

	// Purge the docker container
	if err := pool.Purge(resource); err != nil {
		log.Fatalf("could not purge resource: %s", err)
	}

	os.Exit(code)
}

func TestUser_Table(t *testing.T) {
	s := models.Users.Table()
	if s != "users" {
		t.Error("wrong table name returned: ", s)
	}
}

func createTables(db *sql.DB) error {
	stmt := `CREATE OR REPLACE FUNCTION trigger_set_timestamp()
	RETURNS TRIGGER AS $$
	BEGIN
	  NEW.updated_at = NOW();
	RETURN NEW;
	END;
	$$ LANGUAGE plpgsql;
	
	drop table if exists users cascade;
	
	CREATE TABLE users (
		id SERIAL PRIMARY KEY,
		first_name character varying(255) NOT NULL,
		last_name character varying(255) NOT NULL,
		user_active integer NOT NULL DEFAULT 0,
		email character varying(255) NOT NULL UNIQUE,
		password character varying(60) NOT NULL,
		created_at timestamp without time zone NOT NULL DEFAULT now(),
		updated_at timestamp without time zone NOT NULL DEFAULT now()
	);
	
	CREATE TRIGGER set_timestamp
		BEFORE UPDATE ON users
		FOR EACH ROW
		EXECUTE PROCEDURE trigger_set_timestamp();
	
	drop table if exists remember_tokens;
	
	CREATE TABLE remember_tokens (
		id SERIAL PRIMARY KEY,
		user_id integer NOT NULL REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,
		remember_token character varying(100) NOT NULL,
		created_at timestamp without time zone NOT NULL DEFAULT now(),
		updated_at timestamp without time zone NOT NULL DEFAULT now()
	);
	
	CREATE TRIGGER set_timestamp
		BEFORE UPDATE ON remember_tokens
		FOR EACH ROW
		EXECUTE PROCEDURE trigger_set_timestamp();
	
	drop table if exists tokens;
	
	CREATE TABLE tokens (
		id SERIAL PRIMARY KEY,
		user_id integer NOT NULL REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,
		first_name character varying(255) NOT NULL,
		email character varying(255) NOT NULL,
		token character varying(255) NOT NULL,
		token_hash bytea NOT NULL,
		created_at timestamp without time zone NOT NULL DEFAULT now(),
		updated_at timestamp without time zone NOT NULL DEFAULT now(),
		expiry timestamp without time zone NOT NULL
	);
	
	CREATE TRIGGER set_timestamp
		BEFORE UPDATE ON tokens
		FOR EACH ROW
		EXECUTE PROCEDURE trigger_set_timestamp();`

	_, err := db.Exec(stmt)
	if err != nil {
		return err
	}
	return nil
}

func TestUser_Insert(t *testing.T) {
	id, err := models.Users.Insert(dummyUser)
	if err != nil {
		t.Error("failed to insert user: ", err)
	}

	if id == 0 {
		t.Error("0 returned as id after insert")
	}
}

func TestUser_Get(t *testing.T) {
	u, err := models.Users.Get(1)
	if err != nil {
		t.Error("failed to get user: ", err)
	}

	if u.ID == 0 {
		t.Error("id of returned user is 0")
	}
}

func TestUser_GetAll(t *testing.T) {
	_, err := models.Users.GetAll()
	if err != nil {
		t.Error("failed to get all users: ", err)
	}
}

func TestUser_GetByEmail(t *testing.T) {
	u, err := models.Users.GetByEmail(dummyUser.Email)
	if err != nil {
		t.Error("failed to get user by email: ", err)
	}

	if u.ID == 0 {
		t.Error("id of returned user is 0")
	}
}

func TestUser_Update(t *testing.T) {
	u, err := models.Users.Get(1)
	if err != nil {
		t.Error("failed to get user: ", err)
	}

	u.LastName = "Smith"
	err = u.Update(*u)
	if err != nil {
		t.Error("failed to update the user: ", err)
	}

	u, err = models.Users.Get(1)
	if err != nil {
		t.Error("failed to get user: ", err)
	}

	if u.LastName != "Smith" {
		t.Error("last name not updated in database")
	}
}

func TestUser_PasswordMatches(t *testing.T) {
	u, err := models.Users.Get(1)
	if err != nil {
		t.Error("failed to get user: ", err)
	}

	matches, err := u.PasswordMatches(dummyUser.Password)
	if err != nil {
		t.Error("error checking password match: ", err)
	}

	if !matches {
		t.Error("password does not match when it should")
	}

	matches, err = u.PasswordMatches("123")
	if err != nil {
		t.Error("error checking password match: ", err)
	}

	if matches {
		t.Error("password matches when it should not")
	}
}

func TestUser_ResetPassword(t *testing.T) {
	err := models.Users.ResetPassword(1, "new-password")
	if err != nil {
		t.Error("error resetting password: ", err)
	}

	err = models.Users.ResetPassword(9999, "new-password")
	if err == nil {
		t.Error("did not get an error when trying to reset password for non existing user")
	}
}

func TestUser_Delete(t *testing.T) {
	err := models.Users.Delete(1)
	if err != nil {
		t.Error("error deleting user: ", err)
	}

	_, err = models.Users.Get(1)
	if err == nil {
		t.Error("retrieved user who was supposed to be deleted")
	}
}

func TestToken_Table(t *testing.T) {
	s := models.Tokens.Table()
	if s != "tokens" {
		t.Error("wrong table name returned for tokens: ", s)
	}
}

func TestToken_GenerateToken(t *testing.T) {
	// Insert a user
	id, err := models.Users.Insert(dummyUser)
	if err != nil {
		t.Error("error inserting user: ", err)
	}
	// Get a token
	_, err = models.Tokens.GenerateToken(id, time.Hour*24*365)
	if err != nil {
		t.Error("error generating token: ", err)
	}
}

func TestToken_Insert(t *testing.T) {
	u, err := models.Users.GetByEmail(dummyUser.Email)
	if err != nil {
		t.Error("failed to get user: ", err)
	}

	token, err := models.Tokens.GenerateToken(u.ID, time.Hour*24*365)
	if err != nil {
		t.Error("error generating token: ", err)
	}

	err = models.Tokens.Insert(*token, *u)
	if err != nil {
		t.Error("error inserting token: ", err)
	}
}

func TestToken_GetUserForToken(t *testing.T) {
	token := "ABC"
	_, err := models.Tokens.GetUserForToken(token)
	if err == nil {
		t.Error("error expected but not received when getting user with a bad token")
	}

	u, err := models.Users.GetByEmail(dummyUser.Email)
	if err != nil {
		t.Error("failed to get user: ", err)
	}

	_, err = models.Tokens.GetUserForToken(u.Token.PlainText)
	if err != nil {
		t.Error("failed to get user with valid token: ", err)
	}
}

func TestToken_GetTokensForUser(t *testing.T) {
	tokens, err := models.Tokens.GetTokensForUser(999)
	if err != nil {
		t.Error(err)
	}

	if len(tokens) > 0 {
		t.Error("tokens returned for non existing user")
	}
}

func TestToken_Get(t *testing.T) {
	u, err := models.Users.GetByEmail(dummyUser.Email)
	if err != nil {
		t.Error("failed to get user: ", err)
	}

	_, err = models.Tokens.Get(u.Token.ID)
	if err != nil {
		t.Error("error getting token by ID: ", err)
	}
}

func TestToken_GetByToken(t *testing.T) {
	u, err := models.Users.GetByEmail(dummyUser.Email)
	if err != nil {
		t.Error("failed to get user: ", err)
	}

	_, err = models.Tokens.GetByToken(u.Token.PlainText)
	if err != nil {
		t.Error("error getting token by token: ", err)
	}

	_, err = models.Tokens.GetByToken("xyz")
	if err == nil {
		t.Error("no error getting non existing token")
	}
}

var authData = []struct {
	name          string
	token         string
	email         string
	errorExpected bool
	message       string
}{
	{"invalid", "abcdefghijklmnopqrstuvwxyz", "a@here.com", true, "invalid token accepted as value"},
	{"invalid_length", "abcdefghijklmnopqrstuvwxy", "a@here.com", true, "token of wrong length accepted as value"},
	{"no_user", "abcdefghijklmnopqrstuvwxyz", "a@here.com", true, "no user, but token accepted as value"},
	{"valid", "", dummyUser.Email, false, "valid token reported as invalid"},
}

func TestToken_AuthenticateToken(t *testing.T) {
	for _, tt := range authData {
		token := ""
		if tt.email == dummyUser.Email {
			user, err := models.Users.GetByEmail(tt.email)
			if err != nil {
				t.Error("failed to get user: ", err)
			}
			token = user.Token.PlainText
		} else {
			token = tt.token
		}

		req, _ := http.NewRequest("GET", "/", nil)
		req.Header.Add("Authorization", "Bearer "+token)

		_, err := models.Tokens.AuthenticateToken(req)
		if tt.errorExpected && err == nil {
			t.Errorf("%s: %s", tt.name, tt.message)
		} else if !tt.errorExpected && err != nil {
			t.Errorf("%s: %s - %s", tt.name, tt.message, err)
		} else {
			t.Logf("passed %s", tt.name)
		}
	}
}

func TestToken_Delete(t *testing.T) {
	u, err := models.Users.GetByEmail(dummyUser.Email)
	if err != nil {
		t.Error(err)
	}

	err = models.Tokens.DeleteByToken(u.Token.PlainText)
	if err != nil {
		t.Error("error deleting token: ", err)
	}
}

func TestToken_ExpiredToken(t *testing.T) {
	// Insert a token
	u, err := models.Users.GetByEmail(dummyUser.Email)
	if err != nil {
		t.Error(err)
	}

	token, err := models.Tokens.GenerateToken(u.ID, -time.Hour)
	if err != nil {
		t.Error(err)
	}

	err = models.Tokens.Insert(*token, *u)
	if err != nil {
		t.Error(err)
	}

	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Add("Authorization", "Bearer "+token.PlainText)

	_, err = models.Tokens.AuthenticateToken(req)
	if err == nil {
		t.Error("failed to catch expired token")
	}
}

func TestToken_BadHeader(t *testing.T) {
	req, _ := http.NewRequest("GET", "/", nil)
	_, err := models.Tokens.AuthenticateToken(req)
	if err == nil {
		t.Error("failed to catch missing auth header")
	}

	req, _ = http.NewRequest("GET", "/", nil)
	req.Header.Add("Authorization", "ABC")
	if err == nil {
		t.Error("failed to catch incorrect auth header")
	}

	newUser := User{
		FirstName: "Temp",
		LastName:  "Temp Last",
		Email:     "you@there.com",
		Active:    1,
		Password:  "abc",
	}

	id, err := models.Users.Insert(newUser)
	if err != nil {
		t.Error(err)
	}

	token, err := models.Tokens.GenerateToken(id, 1*time.Hour)
	if err != nil {
		t.Error(err)
	}

	err = models.Tokens.Insert(*token, newUser)
	if err != nil {
		t.Error(err)
	}

	err = models.Users.Delete(id)
	if err != nil {
		t.Error(err)
	}

	// When we delete a user the token should be gone because of cascade foreign key
	req, _ = http.NewRequest("GET", "/", nil)
	req.Header.Add("Authorization", "Bearer "+token.PlainText)

	_, err = models.Tokens.AuthenticateToken(req)
	if err == nil {
		t.Error("failed to catch token for deleted user")
	}
}

func TestToken_DeleteNotExistingToken(t *testing.T) {
	err := models.Tokens.DeleteByToken("abc")
	if err != nil {
		t.Error("error deleting token: ", err)
	}
}

func TestToken_ValidToken(t *testing.T) {
	u, err := models.Users.GetByEmail(dummyUser.Email)
	if err != nil {
		t.Error(err)
	}

	newToken, err := models.Tokens.GenerateToken(u.ID, 24*time.Hour)
	if err != nil {
		t.Error(err)
	}

	err = models.Tokens.Insert(*newToken, *u)
	if err != nil {
		t.Error(err)
	}

	ok, err := models.Tokens.ValidToken(newToken.PlainText)
	if err != nil {
		t.Error("error calling valid token: ", err)
	}
	if !ok {
		t.Error("valid token reported as invalid")
	}

	ok, _ = models.Tokens.ValidToken("abc")
	if ok {
		t.Error("invalid token reported as invalid")
	}

	u, err = models.Users.GetByEmail(dummyUser.Email)
	if err != nil {
		t.Error(err)
	}

	err = models.Tokens.Delete(u.Token.ID)
	if err != nil {
		t.Error(err)
	}

	ok, err = models.Tokens.ValidToken(u.Token.PlainText)
	if err == nil {
		t.Error("no error reported when validating non existing token")
	}
	if ok {
		t.Error("token reported as valid for non existing token")
	}
}
