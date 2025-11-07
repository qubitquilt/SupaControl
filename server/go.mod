module github.com/qubitquilt/supacontrol/server

go 1.24.7

require (
	github.com/golang-jwt/jwt/v5 v5.2.0
	github.com/jmoiron/sqlx v1.3.5
	github.com/labstack/echo/v4 v4.11.4
	github.com/lib/pq v1.10.9
	github.com/qubitquilt/supacontrol/pkg/api-types v0.0.0
	golang.org/x/crypto v0.17.0
	helm.sh/helm/v3 v3.13.3
	k8s.io/api v0.28.4
	k8s.io/apimachinery v0.28.4
	k8s.io/client-go v0.28.4
)

replace github.com/qubitquilt/supacontrol/pkg/api-types => ../pkg/api-types
