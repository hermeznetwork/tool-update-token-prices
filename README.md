# tool-update-token-prices
Update token prices for a specific hermez node from a specific price updater service

## Requirements

- Hermez node postgres database with tokens registered in table `Token`
- Price updater service
- Flags have priority over Environment variables

## How it works

- Load all tokens from table `token`
- Get token prices from price updater service
- Update prices on table `token`

## How to use

Compile it from the root dir of this repo:

```bash
go build -o update-price ./...
```

Set the env variables and run:

```bash
POSTGRES_HOST="" POSTGRES_PORT="" POSTGRES_USER="" POSTGRES_PASSWORD="" POSTGRES_DATABASE="" PRICE_UPDATER_URL="" PRICE_UPDATER_API_KEY="" ./update-price
```

OR

Set the values from the flags

```bash
./update-price -POSTGRES_HOST="" -POSTGRES_PORT="" -POSTGRES_USER="" -POSTGRES_PASSWORD="" -POSTGRES_DATABASE="" -PRICE_UPDATER_URL="" -PRICE_UPDATER_API_KEY=""
```
