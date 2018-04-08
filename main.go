package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"

	"github.com/go-redis/redis"
	. "github.com/logrusorgru/aurora"
)

var indexT *template.Template

type Equipment struct {
	Name  string `json:"name"`
	Model string `json:"model"`
}

type router struct {
	redis *redis.Client
}

func init() {
	indexT = template.Must(template.ParseFiles("template.html"))
}

func CreateConnection() (*redis.Client, error) {
	config, err := parseConfig()
	if err != nil {
		return nil, err
	}
	redis, err := connectToRedis(config)
	if err != nil {
		return nil, err
	}
	return redis, nil
}

func (e *Equipment) GetJSONFromClient(r *http.Request) error {
	err := json.NewDecoder(r.Body).Decode(&e)
	if err != nil {
		return err
	}
	defer r.Body.Close()
	fmt.Fprintln(os.Stdout, "Got from client >", Magenta(e))
	return nil
}

func GetKeysValuesFromDB(redis *redis.Client) ([]string, []string, error) {
	var cursor uint64
	var keys []string
	var values []string

	keys, cursor, err := redis.Scan(cursor, "", 10).Result()
	if err != nil {
		return nil, nil, err
	}

	for _, key := range keys {
		value, err := redis.Get(key).Result()
		if err != nil {
			return nil, nil, err
		}
		values = append(values, value)
		fmt.Fprintf(os.Stdout,
			"Got from DB: Key > %s, Value > %s ", Gray(key), Gray(value))
	}
	return keys, values, nil
}

func (router *router) GetEquipments() ([]Equipment, error) {
	keys, values, err := GetKeysValuesFromDB(router.redis)
	if err != nil {
		return nil, err
	}
	var result []Equipment
	for i, _ := range keys {
		item := Equipment{
			Name:  values[i],
			Model: keys[i],
		}
		result = append(result, item)
	}
	return result, nil
}

func (e *Equipment) AddToRedis(redis *redis.Client) error {
	err := redis.Append(e.Model, e.Name).Err()
	if err != nil {
		return err
	}
	return nil
}

func (router *router) HandleIndex(w http.ResponseWriter, r *http.Request) {
	items, err := router.GetEquipments()
	if err != nil {
		if err != nil {
			fmt.Fprintln(os.Stderr, "ERROR GetEquipments >", Red(err))
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
	err = indexT.Execute(w, map[string]interface{}{
		"items": items,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR template execute >", Red(err))
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (router *router) HandleAddEquipment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-type", "application/json")

	var e Equipment

	err := e.GetJSONFromClient(r)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR GetJSONFromClient >", Red(err))
		w.WriteHeader(http.StatusInternalServerError)
	}
	err = e.AddToRedis(router.redis)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR AddToRedis >", Red(err))
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (router *router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	method := r.Method
	fmt.Printf("Processing request %s %s\n", method, path)
	switch {
	case method == "GET" && path == "/":
		router.HandleIndex(w, r)
	case method == "POST" && path == "/api/v1/equipments/":
		router.HandleAddEquipment(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func main() {
	redis, err := CreateConnection()
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR CreateConnection >", Red(err))
		os.Exit(1)
	} else {
		fmt.Fprintln(os.Stdout, Green("Connection established"))
	}

	router := &router{
		redis: redis,
	}

	http.ListenAndServe(":8080", router)
}
