# User Management
User management (UM) is defined as the effective management of users giving them access to systems.

# Feature
* CRUD API
* Authentication
* Authorization
* CORS


# Technologies
* [Gin](https://github.com/gin-gonic/gin)
* [MongoDB](https://www.mongodb.com)
* [Redis](https://redis.io)

# Set up
* Create file .env
* Set MongoDB URI and DB
  - PORT = "8585" or your port
  - MONGO_HOST = "your host/ localhost:27017"
  - MONGO_UM_DB_NAME = "your db name"
  - REDIS_HOST = "your redis host"
  - SECRET_KEY = "your secret key"

# Run
* `go mod download` for download dependencies
* `go run main.go`
* `nodemon --exec go run main.go --signal SIGTERM` for run with nodemon


