package main

import (
	"log"
	"net/http"

	"github.com/jayreddy040-510/receipt_processor/internal/app"
	"github.com/jayreddy040-510/receipt_processor/internal/config"
	"github.com/jayreddy040-510/receipt_processor/internal/db"

	"github.com/go-chi/chi"
)

func main() {
	// load config
	log.Println("Loading configuration...")
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
		return
	}
	log.Println("Configuration loaded!")

	// init and check connection to db
	log.Println("Initializing DB client and testing connection...")
	db := db.NewRedisStore(cfg)
	if err := db.CheckConnection(); err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
	log.Println("Successfully connected to DB!")

	// init shared resources struct
	a := &app.App{
		Db: db,
	}

	// init router, connect routes to handlers
	r := chi.NewRouter()
	r.Route("/receipts", func(r chi.Router) {
		r.Post("/process", a.ProcessReceiptHandler)
		r.Get("/{id}/points", a.GetPointsHandler)
	})

	// boot up server
	log.Printf("Starting server on :%s...", cfg.ServerPort)
	if err := http.ListenAndServe(":"+cfg.ServerPort, r); err != nil {
		log.Fatalf("Server exited: %v", err)
	}

}
