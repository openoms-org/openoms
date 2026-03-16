module github.com/openoms-org/openoms/apps/api-server

go 1.25.0

toolchain go1.25.7

require (
	github.com/aws/aws-sdk-go-v2 v1.41.4
	github.com/aws/aws-sdk-go-v2/config v1.32.12
	github.com/aws/aws-sdk-go-v2/credentials v1.19.12
	github.com/aws/aws-sdk-go-v2/service/s3 v1.96.0
	github.com/caarlos0/env/v11 v11.4.0
	github.com/getsentry/sentry-go v0.43.0
	github.com/go-chi/chi/v5 v5.2.5
	github.com/golang-jwt/jwt/v5 v5.2.2
	github.com/google/uuid v1.6.0
	github.com/gorilla/websocket v1.5.3
	github.com/jackc/pgx/v5 v5.8.0
	github.com/microcosm-cc/bluemonday v1.0.27
	github.com/openoms-org/openoms/packages/allegro-go-sdk v0.0.0-20260213093925-f69d292073cb
	github.com/openoms-org/openoms/packages/amazon-sp-sdk v0.0.0-20260213093925-f69d292073cb
	github.com/openoms-org/openoms/packages/btp-go-sdk v0.0.0-00010101000000-000000000000
	github.com/openoms-org/openoms/packages/dhl-go-sdk v0.0.0-20260213093925-f69d292073cb
	github.com/openoms-org/openoms/packages/dpd-go-sdk v0.0.0-20260213093925-f69d292073cb
	github.com/openoms-org/openoms/packages/ebay-go-sdk v0.0.0-20260213093925-f69d292073cb
	github.com/openoms-org/openoms/packages/erli-go-sdk v0.0.0-20260213093925-f69d292073cb
	github.com/openoms-org/openoms/packages/fakturownia-go-sdk v0.0.0-20260213093925-f69d292073cb
	github.com/openoms-org/openoms/packages/fedex-go-sdk v0.0.0-20260213093925-f69d292073cb
	github.com/openoms-org/openoms/packages/gls-go-sdk v0.0.0-20260213093925-f69d292073cb
	github.com/openoms-org/openoms/packages/infakt-go-sdk v0.0.0-20260213093925-f69d292073cb
	github.com/openoms-org/openoms/packages/inpost-go-sdk v0.0.0-20260213093925-f69d292073cb
	github.com/openoms-org/openoms/packages/iof-parser v0.0.0-20260213093925-f69d292073cb
	github.com/openoms-org/openoms/packages/kaufland-go-sdk v0.0.0-20260213093925-f69d292073cb
	github.com/openoms-org/openoms/packages/ksef-go-sdk v0.0.0-20260213093925-f69d292073cb
	github.com/openoms-org/openoms/packages/mirakl-go-sdk v0.0.0-20260213093925-f69d292073cb
	github.com/openoms-org/openoms/packages/olx-go-sdk v0.0.0-20260213093925-f69d292073cb
	github.com/openoms-org/openoms/packages/order-engine v0.0.0-20260213093925-f69d292073cb
	github.com/openoms-org/openoms/packages/orlen-paczka-go-sdk v0.0.0-20260213093925-f69d292073cb
	github.com/openoms-org/openoms/packages/poczta-polska-go-sdk v0.0.0-20260213093925-f69d292073cb
	github.com/openoms-org/openoms/packages/prestashop-go-sdk v0.0.0-20260213093925-f69d292073cb
	github.com/openoms-org/openoms/packages/shoper-go-sdk v0.0.0-20260213093925-f69d292073cb
	github.com/openoms-org/openoms/packages/shopify-go-sdk v0.0.0-20260213093925-f69d292073cb
	github.com/openoms-org/openoms/packages/smsapi-go-sdk v0.0.0-20260213093925-f69d292073cb
	github.com/openoms-org/openoms/packages/ups-go-sdk v0.0.0-20260213093925-f69d292073cb
	github.com/openoms-org/openoms/packages/wfirma-go-sdk v0.0.0-20260213093925-f69d292073cb
	github.com/openoms-org/openoms/packages/woocommerce-go-sdk v0.0.0-20260213093925-f69d292073cb
	github.com/pquerna/otp v1.5.0
	github.com/redis/go-redis/v9 v9.18.0
	github.com/stretchr/testify v1.11.1
	github.com/stripe/stripe-go/v82 v82.5.1
	golang.org/x/crypto v0.49.0
	golang.org/x/text v0.35.0
)

require (
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.4 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.20 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.20 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.20 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.6 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.9.8 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.20 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.19.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.0.8 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.30.13 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.35.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.41.9 // indirect
	github.com/aws/smithy-go v1.24.2 // indirect
	github.com/aymerick/douceur v0.2.0 // indirect
	github.com/boombuler/barcode v1.0.1-0.20190219062509-6c824513bacc // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/gorilla/css v1.0.1 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/net v0.51.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/openoms-org/openoms/packages/wfirma-go-sdk => ../../packages/wfirma-go-sdk

replace github.com/openoms-org/openoms/packages/infakt-go-sdk => ../../packages/infakt-go-sdk

replace github.com/openoms-org/openoms/packages/shoper-go-sdk => ../../packages/shoper-go-sdk

replace github.com/openoms-org/openoms/packages/prestashop-go-sdk => ../../packages/prestashop-go-sdk

replace github.com/openoms-org/openoms/packages/shopify-go-sdk => ../../packages/shopify-go-sdk

replace github.com/openoms-org/openoms/packages/btp-go-sdk => ../../packages/btp-go-sdk
