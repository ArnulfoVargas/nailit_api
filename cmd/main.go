package main

import (
	"database/sql"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	// "github.com/joho/godotenv"
	_ "github.com/go-sql-driver/mysql"
)

type Server struct {
	db *sql.DB
	port string
	app *fiber.App
}

func main() {
	// godotenv.Load();

	server := Server{}
	server.openDB()
	server.port = ":"+os.Getenv("PORT")

	server.app = fiber.New()
	server.handleControllers()

	log.Fatal(server.app.Listen(server.port))
}

func (server *Server) openDB() {
	user := os.Getenv("DB_USER")
	pass := os.Getenv("DB_PASSWORD")
	host := os.Getenv("DB_HOST")
	name := os.Getenv("DB_NAME")
	
	builder := strings.Builder{}

	builder.WriteString(user)
	builder.WriteByte(':')
	builder.WriteString(pass)
	builder.WriteByte('@')
	builder.WriteString("tcp(")
	builder.WriteString(host)
	builder.WriteString(":3306)/")
	builder.WriteString(name)

	db, err := sql.Open("mysql", builder.String())
	if err != nil {
		println(builder.String())
		panic(err)
	}

	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	server.db = db;
}