module github.com/stitts-dev/dfs-sim/services/realtime-service

go 1.21

require (
	github.com/gin-gonic/gin v1.9.1
	github.com/google/uuid v1.3.0
	github.com/gorilla/websocket v1.5.0
	github.com/redis/go-redis/v9 v9.0.5
	github.com/sirupsen/logrus v1.9.3
	github.com/stitts-dev/dfs-sim/shared v0.0.0
	gorm.io/datatypes v1.2.0
	gorm.io/driver/postgres v1.5.2
	gorm.io/gorm v1.25.4
)

require (
	github.com/lib/pq v1.10.9
	github.com/spf13/viper v1.16.0
)

replace github.com/stitts-dev/dfs-sim/shared => ../../shared