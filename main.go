package main

import(
    "os"
    "log"
    "fmt"
    "math"
    "time"
    "strings"
    "unicode"
    "strconv"
    "net/http"
    "encoding/json"

    "github.com/google/uuid"
)

type item struct {
    ShortDescription    string `json:"shortDescription"`
    Price               string `json:"price"`
}

type receipt struct {
    Retailer        string `json:"retailer"`
    PurchaseDate    string `json:"purchaseDate"`
    PurchaseTime    string `json:"purchaseTime"`
    Items           []item `json:"items"`
    Total           string `json:"total"`
}

func parseDollarAsStringInput(amt string) (float64, error) {

    amt = strings.ReplaceAll(amt, ",", "") // sanitize input if commas

    for pos, char := range amt {
        if !unicode.IsDigit(char) && char != '.' {
            return 0, fmt.Errorf("Error parsing dollar amt: invalid character")
        }
        if char == '.' {
            if len(amt) - pos - 1 != 2 {
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
    purchaseTimeAndDate, err := time.Parse("2006-01-02 15:04", dateString + " " + timeString)
    if err != nil {
        return time.Time{}, fmt.Errorf("Error parsing purchaseTimeAndDate: %v", err)
    }
    if purchaseTimeAndDate.After(time.Now()) {
        return time.Time{}, fmt.Errorf("Error parsing purchaseTimeAndDate: future time given (%v)", purchaseTimeAndDate)
    }
    return purchaseTimeAndDate, nil
}

func receiptRetailerHelper(retailer string) int {
    var count int
    for _, char := range retailer {
        if unicode.IsLetter(char) || unicode.IsDigit(char) {
            count++
        }
    }
    return count
}

func receiptTotalHelper(total string) (int, error) {
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

func receiptItemsHelper(items []item) int {
    var points int
    for _, item := range items {
        if trimmed := strings.Trim(item.ShortDescription, " "); len(trimmed) % 3 == 0 {
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

func receiptPurchaseDateHelper(date string) (int, error) {
    dayValue, err := parseDateAsStringInput(date) 
    if err != nil {
        return 0, err
    }
    if dayValue % 2 != 0 {
        return 6, nil
    }
    return 0, nil
}

func receiptPurchaseTimeHelper(timeString, dateString string) (int, error) {
    purchaseTimeAndDate, err := parseTimeAsStringInput(timeString, dateString)
    if err != nil {
        return 0, err
    }
    // use HHMM format because easy int format to compare times, rather than using
    // time.Parse() and time.After() and time.Before() several times
    purchaseHHMM := purchaseTimeAndDate.Hour() * 100 + purchaseTimeAndDate.Minute()

    if purchaseHHMM > 1400 && purchaseHHMM < 1600 {
        return 10, nil
    }

    return 0, nil
}

func processReceiptHandler(w http.ResponseWriter, r *http.Request) {
    var rec receipt
    var pointsTotal int
    err := json.NewDecoder(r.Body).Decode(&rec)
    r.Body.Close() // maybe should use defer here?
    if err != nil {
        log.Printf("Error decoding request body: %v", err)
        http.Error(w, "The receipt is invalid", http.StatusBadRequest)
        return
    }

    pointsTotal += receiptRetailerHelper(rec.Retailer) 
    pointsFromReceiptTotal, err := receiptTotalHelper(rec.Total)
    if err != nil {
        log.Println(err)
        http.Error(w, "The receipt is invalid", http.StatusBadRequest)
        return
    }
    pointsTotal += pointsFromReceiptTotal
    pointsTotal += (len(rec.Items)/2) * 5 // dont need a helper for this (5 points per pair of items)
    pointsTotal += receiptItemsHelper(rec.Items)
    pointsFromPurchaseDateDay, err := receiptPurchaseDateHelper(rec.PurchaseDate)
    if err != nil {
        log.Println(err)
        http.Error(w, "The receipt is invalid", http.StatusBadRequest)
        return
    }
    pointsTotal += pointsFromPurchaseDateDay
    pointsFromPurchaseTimeHour, err :=  receiptPurchaseTimeHelper(rec.PurchaseTime, rec.PurchaseDate)
    if err != nil {
        log.Println(err)
        http.Error(w, "The receipt is invalid", http.StatusBadRequest)
        return
    }
    pointsTotal += pointsFromPurchaseTimeHour
    
}

func main() {
}