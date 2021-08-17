package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type config struct {
	postgresHost     string
	postgresPort     string
	postgresUser     string
	postgresPassword string
	postgresDatabase string

	priceUpdaterURL string
}

type token struct {
	ID     int      `db:"token_id"`
	Symbol string   `db:"symbol"`
	USD    *float64 `db:"usd"`
}

type tokenPrice struct {
	ID     int     `json:"id"`
	Symbol string  `json:"symbol"`
	USD    float64 `json:"USD"`
}

func main() {

	cfg := getConfig()

	db, err := newDB(cfg)
	if err != nil {
		fmt.Println("failed to create db connection")
		panic(err)
	}

	tokens, err := getTokens(db)
	if err != nil {
		fmt.Println("failed to get tokens")
		panic(err)
	}

	prices, err := getPrices(cfg)
	if err != nil {
		fmt.Println("failed to get token prices")
		panic(err)
	}

	for _, token := range tokens {
		if tp, ok := prices[token.ID]; ok {
			err := updateToken(db, token.ID, tp.USD)
			if err != nil {
				fmt.Printf("ERROR: failed to update price for token %d %s, err: %v\n", tp.ID, tp.Symbol, err)
				continue
			}

			tPrice := "null"
			if token.USD != nil {
				tPrice = fmt.Sprintf("%f", *token.USD)
			}

			fmt.Printf("Token %d %s price updated from %s to %f\n", tp.ID, tp.Symbol, tPrice, tp.USD)
		}
	}
}

func parseConfigValue(name, envValue string, flValue *string) string {
	var v string

	if flValue != nil && len(*flValue) != 0 {
		v = strings.TrimSpace(*flValue)
	} else {
		v = strings.TrimSpace(envValue)
	}

	if len(strings.TrimSpace(v)) == 0 {
		panic(fmt.Sprintf("config required: %s", name))
	}

	return v
}

func getConfig() config {

	postgresHostFromEnv := os.Getenv("POSTGRES_HOST")
	postgresHostFromFlag := flag.String("POSTGRES_HOST", "", "postgres server host")

	postgresPortFromEnv := os.Getenv("POSTGRES_PORT")
	postgresPortFromFlag := flag.String("POSTGRES_PORT", "", "postgres server port")

	postgresUserFromEnv := os.Getenv("POSTGRES_USER")
	postgresUserFromFlag := flag.String("POSTGRES_USER", "", "postgres server user")

	postgresPasswordFromEnv := os.Getenv("POSTGRES_PASSWORD")
	postgresPasswordFromFlag := flag.String("POSTGRES_PASSWORD", "", "postgres server password")

	postgresDBFromEnv := os.Getenv("POSTGRES_DATABASE")
	postgresDBFromFlag := flag.String("POSTGRES_DATABASE", "", "postgres server database")

	priceUpdaterURLFromEnv := os.Getenv("PRICE_UPDATER_URL")
	priceUpdaterURLFromFlag := flag.String("PRICE_UPDATER_URL", "", "price updater service url")

	flag.Parse()

	return config{
		postgresHost:     parseConfigValue("POSTGRES_HOST", postgresHostFromEnv, postgresHostFromFlag),
		postgresPort:     parseConfigValue("POSTGRES_PORT", postgresPortFromEnv, postgresPortFromFlag),
		postgresUser:     parseConfigValue("POSTGRES_USER", postgresUserFromEnv, postgresUserFromFlag),
		postgresPassword: parseConfigValue("POSTGRES_PASSWORD", postgresPasswordFromEnv, postgresPasswordFromFlag),
		postgresDatabase: parseConfigValue("POSTGRES_DATABASE", postgresDBFromEnv, postgresDBFromFlag),

		priceUpdaterURL: parseConfigValue("PRICE_UPDATER_URL", priceUpdaterURLFromEnv, priceUpdaterURLFromFlag),
	}
}

func newDB(cfg config) (*sqlx.DB, error) {
	return sqlx.Connect("postgres", fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", cfg.postgresHost, cfg.postgresPort, cfg.postgresUser, cfg.postgresPassword, cfg.postgresDatabase))
}

func getTokens(db *sqlx.DB) ([]token, error) {
	tokens := []token{}
	err := db.Select(&tokens, "SELECT token_id, symbol, usd FROM TOKEN ORDER BY token_id")
	if err != nil {
		return nil, err
	}
	return tokens, nil
}

func updateToken(db *sqlx.DB, id int, price float64) error {
	_, err := db.Exec("UPDATE TOKEN SET USD_UPDATE = NOW(), USD = $2 WHERE TOKEN_ID = $1", id, price)
	return err
}

func getPrices(cfg config) (map[int]tokenPrice, error) {

	u, err := url.Parse(cfg.priceUpdaterURL)
	if err != nil {
		return nil, err
	}
	u.Path = "v1/tokens"

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Origin", "any")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid status code %d", res.StatusCode)
	}

	var data struct {
		Tokens []tokenPrice `json:"tokens"`
	}
	if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		return nil, err
	}

	tp := map[int]tokenPrice{}

	fmt.Printf("Token prices found: %d\n", len(data.Tokens))
	for _, tokenPrice := range data.Tokens {
		tp[tokenPrice.ID] = tokenPrice
		fmt.Printf("%d %s %f\n", tokenPrice.ID, tokenPrice.Symbol, tokenPrice.USD)
	}

	return tp, nil
}
