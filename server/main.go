package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type item struct {
	ShortDescription string `json:"shortDescription"`
	Price            string `json:"price"`
}

type receipt struct {
	Retailer     string `json:"retailer"`
	PurchaseDate string `json:"purchaseDate"`
	PurchaseTime string `json:"purchaseTime"`
	Items        []item `json:"items"`
	Total        string `json:"total"`
}

type config struct {
	serverPort       string
	redisAddr        string
	dbTimeoutInMs    time.Duration
	redisTTLInSec    time.Duration
	maxDBConnRetries int
}

type redisStore struct {
	client *redis.Client
	config config
}

type App struct {
	db *redisStore
}

func NewRedisStore(config config) *redisStore {
	return &redisStore{
		client: redis.NewClient(&redis.Options{
			Addr: config.redisAddr,
		}),
		config: config,
	}
}

func (rs *redisStore) checkConnection() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*time.Duration(rs.config.dbTimeoutInMs))
	defer cancel()

	return rs.client.Ping(ctx).Err()
}

func (rs *redisStore) getKey(key string) (string, error) {
	// see design decision in setKey below
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*time.Duration(rs.config.dbTimeoutInMs))
	defer cancel()

	for i := 0; i < rs.config.maxDBConnRetries; i++ {
		storedValue, err := rs.client.Get(ctx, key).Result()
		if err == context.DeadlineExceeded {
			log.Printf("Connection to DB timed out, attempting retry, retries attempted: %v", i)
			continue
		} else if err == redis.Nil {
			return "", fmt.Errorf("Key does not exist in database: %v", err)
		} else if err != nil {
			return "", fmt.Errorf("Error getting key from database: %v", err)
		} else {
			return storedValue, nil
		}
	}
	return "", fmt.Errorf("Error connecting to DB: %v. Max retries attempted.", context.DeadlineExceeded)
}

func (rs *redisStore) setKey(key, value string) error {
	// design decision: pass in ctx with timeout to setter from main or define here?
	// because only 1 way we plan on setting and don't plan on changing cancel/timeout logic,
	// i think it's fine to init ctx here
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*time.Duration(rs.config.dbTimeoutInMs))
	defer cancel()

	for i := 0; i < rs.config.maxDBConnRetries; i++ {
		err := rs.client.Set(ctx, key, value, time.Second*time.Duration(rs.config.redisTTLInSec)).Err()
		if err == context.DeadlineExceeded {
			log.Printf("Connection to DB timed out, attempting retry, retries attempted: %v", i)
			continue
		} else if err != nil {
			return fmt.Errorf("Error setting key in database: %v", err)
		} else {
			return err
		}
	}
	return fmt.Errorf("Error connecting to DB: %v. Max retries attempted.", context.DeadlineExceeded)
}

func isValidUUIDv4(s string) (bool, error) {
	// validate incoming URL id before allowing to touch DB
	u, err := uuid.Parse(s)
	if err != nil {
		return false, fmt.Errorf("Invalid UUIDv4: %v", err)
	}
	// checks if UUIDv4
	if u.Version() != uuid.Version(4) {
		return false, fmt.Errorf("Invalid UUIDv4: %v", err)
    }
	return true, nil
}

func parseDollarAsStringInput(amt string) (float64, error) {
	// accept dollar amt as string, return float64 if valid amt
	// design decision: allow for prices without decimal? (should we allow for 36 == $36)?
	// design decision: allow for leading 0's? strconv.ParseFloat() can handle: should we allow for 05.01 == $5.01?
	amt = strings.ReplaceAll(amt, ",", "") // sanitize input if commas

	for pos, char := range amt {
		if !unicode.IsDigit(char) && char != '.' {
			return 0, fmt.Errorf("Error parsing dollar amt: invalid character")
		}
		if char == '.' {
			if len(amt)-pos-1 != 2 {
				return 0, fmt.Errorf("Error parsing dollar amt: incorrect value")
			}
		}
	}

	f, err := strconv.ParseFloat(amt, 64)
	if err != nil {
		return 0, fmt.Errorf("Error parsing dollar amt: %v", err)
	}
	return f, nil
}

func parseDateAsStringInput(dateString string) (int, error) {
	// determine if valid date and return day number to caller
	purchaseDate, err := time.Parse("2006-01-02", dateString)
	if err != nil {
		return -1, fmt.Errorf("Error parsing purchaseDate: %v", err)
	}

	if purchaseDate.After(time.Now()) {
		return -1, fmt.Errorf("Error parsing purchaseDate: future date given (%v)", purchaseDate)
	}
	return purchaseDate.Day(), nil

}

func parseTimeAsStringInput(timeString, dateString string) (time.Time, error) {
	// determine if valid time and return time.Time object
	// need date to see if time given is invalid (could be present day and time after current time)
	purchaseTimeAndDate, err := time.Parse("2006-01-02 15:04", dateString+" "+timeString)
	if err != nil {
		return time.Time{}, fmt.Errorf("Error parsing purchaseTimeAndDate: %v", err)
	}
	if purchaseTimeAndDate.After(time.Now()) {
		return time.Time{}, fmt.Errorf("Error parsing purchaseTimeAndDate: future time given (%v)", purchaseTimeAndDate)
	}
	return purchaseTimeAndDate, nil
}

func calculateRetailerPoints(retailer string) int {
	var count int
	for _, char := range retailer {
		if unicode.IsLetter(char) || unicode.IsDigit(char) {
			count++
		}
	}
	return count
}

func calculateReceiptTotalPoints(total string) (int, error) {
	var points int
	receiptTotalAsFloat, err := parseDollarAsStringInput(total) // returns dollar amt as float64
	if err != nil {
		return 0, err
	}
	if receiptTotalAsFloat == math.Floor(receiptTotalAsFloat) {
		points += 50
	}
	if checkMultipleStatus := receiptTotalAsFloat * 4; checkMultipleStatus == math.Floor(checkMultipleStatus) {
		points += 25
	}

	return points, nil
}

func calculatePointsFromItems(items []item) int {
	var points int
	for _, item := range items {
		if trimmed := strings.Trim(item.ShortDescription, " "); len(trimmed)%3 == 0 {
			// would be cleaner to perform each operation and save to a new variable;
			// but, unnecessary memory allocations inside of a for loop can be expensive?
			// strings.ReplaceAll() is to sanitize the string price input
			f, err := parseDollarAsStringInput(item.Price)
			if err != nil {
				log.Printf("Error processing Item: %+v. %v", item, err)
				continue // design decision: return error to parent func here or continue?
			}
			points += int(math.Ceil(f * 0.2)) // math.Ceil returns a float
		}
	}
	return points
}

func calculatePurchaseDatePoints(date string) (int, error) {
	dayValue, err := parseDateAsStringInput(date)
	if err != nil {
		return 0, err
	}
	if dayValue%2 != 0 {
		return 6, nil
	}
	return 0, nil
}

func calculatePurchaseTimePoints(timeString, dateString string) (int, error) {
	purchaseTimeAndDate, err := parseTimeAsStringInput(timeString, dateString)
	if err != nil {
		return 0, err
	}
	// use HHMM format because easy int format to compare times, rather than using
	// time.Parse() and time.After() and time.Before() several times
	purchaseHHMM := purchaseTimeAndDate.Hour()*100 + purchaseTimeAndDate.Minute()

	if purchaseHHMM > 1400 && purchaseHHMM < 1600 {
		return 10, nil
	}

	return 0, nil
}

func (a *App) processReceiptHandler(w http.ResponseWriter, r *http.Request) {
	var rec receipt
	var pointsTotal int
	err := json.NewDecoder(r.Body).Decode(&rec)
	defer r.Body.Close()
	if err != nil {
		log.Printf("Error decoding request body: %v", err)
		http.Error(w, "The receipt is invalid", http.StatusBadRequest)
		return
	}

	pointsTotal += calculateRetailerPoints(rec.Retailer)
	pointsFromReceiptTotal, err := calculateReceiptTotalPoints(rec.Total)
	if err != nil {
		log.Println(err)
		http.Error(w, "The receipt is invalid", http.StatusBadRequest)
		return
	}
	pointsTotal += pointsFromReceiptTotal
	pointsTotal += (len(rec.Items) / 2) * 5 // dont need a helper for this (5 points per pair of items)
	pointsTotal += calculatePointsFromItems(rec.Items)
	pointsFromPurchaseDateDay, err := calculatePurchaseDatePoints(rec.PurchaseDate)
	if err != nil {
		log.Println(err)
		http.Error(w, "The receipt is invalid", http.StatusBadRequest)
		return
	}
	pointsTotal += pointsFromPurchaseDateDay
	pointsFromPurchaseTimeHour, err := calculatePurchaseTimePoints(rec.PurchaseTime, rec.PurchaseDate)
	if err != nil {
		log.Println(err)
		http.Error(w, "The receipt is invalid", http.StatusBadRequest)
		return
	}
	pointsTotal += pointsFromPurchaseTimeHour
	pointsTotalAsString := strconv.Itoa(pointsTotal)
	uuidString := uuid.New().String()
	err = a.db.setKey(uuidString, pointsTotalAsString)
	if err != nil {
		log.Printf("Error setting DB key-value pair: %v", err)
		http.Error(w, "The receipt is invalid", http.StatusBadRequest)
		return
	}
	log.Printf("id: %s, pts: %d", uuidString, pointsTotal)
	responseToClient := map[string]string{
		"id": uuidString,
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(responseToClient); err != nil {
		log.Printf("Error encoding client response: %v", err)
		http.Error(w, "The receipt is invalid", http.StatusBadRequest)
	}
	return
}

func (a *App) getPointsHandler(w http.ResponseWriter, r *http.Request) {
	receiptId := chi.URLParam(r, "id")
	if ok, err := isValidUUIDv4(receiptId); !ok {
		log.Println(err)
		http.Error(w, "No receipt found for that id", http.StatusNotFound)
		return
	}
	pointsValue, err := a.db.getKey(receiptId)
	if err != nil {
		log.Println(err)
		http.Error(w, "No receipt found for that id", http.StatusNotFound)
		return
	}
	pointsValueAsInt, err := strconv.Atoi(pointsValue)
	if err != nil {
		log.Printf("Error converting points string to int: %v", err)
		http.Error(w, "No receipt found for that id", http.StatusNotFound)
		return
	}
	responseToClient := map[string]int{
		"points": pointsValueAsInt,
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(responseToClient); err != nil {
		log.Printf("Error encoding client response: %v", err)
		http.Error(w, "No receipt found for that id", http.StatusNotFound)
	}
	return
}

func loadConfig() (config, error) {
	// design decision: return config or *config? since main functionality of config is
	// to read it and not write to it, decided to return struct
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "redis:6379"
	}
	serverPort := os.Getenv("SERVER_PORT")
	if serverPort == "" {
		serverPort = "8080"
	}

	// strconv will throw error if os.Getenv("FOO") returns "" - can catch early
	dbTimeoutInMs, err := strconv.Atoi(os.Getenv("DB_TIMEOUT_IN_MS"))
	if err != nil {
		return config{}, fmt.Errorf("Error converting DB_TIMEOUT env to int: %v", err)
	}

	redisTTLInSec, err := strconv.Atoi(os.Getenv("REDIS_TTL_IN_S"))
	if err != nil {
		return config{}, fmt.Errorf("Error converting REDIS_TTL env to int: %v", err)
	}

	maxDBConnRetries, err := strconv.Atoi(os.Getenv("MAX_DB_CONN_RETRIES"))
	if err != nil {
		return config{}, fmt.Errorf("Error converting MAX_DB_CONN_RETRIES env to int: %v", err)
	}

	appConfig := config{
		serverPort:       serverPort,
		redisAddr:        redisAddr,
		dbTimeoutInMs:    time.Millisecond * time.Duration(dbTimeoutInMs),
		redisTTLInSec:    time.Second * time.Duration(redisTTLInSec),
		maxDBConnRetries: maxDBConnRetries,
	}
	return appConfig, nil
}

func main() {
	// load config
	log.Println("Loading configuration...")
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
		return
	}
	log.Println("Configuration loaded!")

	// init and check connection to db
	log.Println("Initializing DB client and testing connection...")
	db := NewRedisStore(cfg)
	if err := db.checkConnection(); err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
	log.Println("Successfully connected to DB!")

	// init shared resources struct
	a := &App{
		db: db,
	}

	// init router, connect routes to handlers
	r := chi.NewRouter()
	r.Route("/receipts", func(r chi.Router) {
		r.Post("/process", a.processReceiptHandler)
		r.Get("/{id}/points", a.getPointsHandler)
	})

	// boot up server
	log.Printf("Starting server on :%s...", cfg.serverPort)
	if err := http.ListenAndServe(":"+cfg.serverPort, r); err != nil {
		log.Fatalf("Server exited: %v", err)
	}

}
