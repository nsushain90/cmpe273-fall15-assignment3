package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	//"bytes"
	//uber "github.com/r-medina/go-uber"
	"github.com/jmoiron/jsonq"
)

// to convert a float number to a string
func FloatToString(input_num float64) string {
	return strconv.FormatFloat(input_num, 'f', 6, 64)
}

func getProduct(lat, lng float64) (productId string) {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", "https://sandbox-api.uber.com/v1/products?latitude="+FloatToString(lat)+"&longitude="+FloatToString(lng), nil)
	req.Header.Add("Authorization", "Token 7QTVuOgroe-PWmAXhGiQUuzhVYaKR-DrdKq9uaYd")
	res, _ := client.Do(req)

	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		log.Fatal(err)
	}

	if res.StatusCode != 200 {
		log.Fatal("Unexpected status code", res.StatusCode)
	}
	data := map[string]interface{}{}
	dec := json.NewDecoder(strings.NewReader(string(body)))
	err = dec.Decode(&data)
	if err != nil {
		fmt.Println(err)
	}
	jq := jsonq.NewQuery(data)

	productId, err = jq.String("products", "0", "product_id")
	if err != nil {
		fmt.Println(err)
	}
	return
}

// Function calls Uber price estimate API and gets price time and duration estimation
func uberPrice(lat1, lng1, lat2, lng2 float64) (price, duration, distance float64) {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", "https://sandbox-api.uber.com/v1/estimates/price?start_latitude="+FloatToString(lat1)+"&start_longitude="+FloatToString(lng1)+"&end_latitude="+FloatToString(lat2)+"&end_longitude="+FloatToString(lng2), nil)
	req.Header.Add("Authorization", "Token 7QTVuOgroe-PWmAXhGiQUuzhVYaKR-DrdKq9uaYd")
	res, _ := client.Do(req)

	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		log.Fatal(err)
	}

	if res.StatusCode != 200 {
		log.Fatal("Unexpected status code", res.StatusCode)
	}
	data := map[string]interface{}{}
	dec := json.NewDecoder(strings.NewReader(string(body)))
	err = dec.Decode(&data)
	if err != nil {
		fmt.Println(err)
	}
	jq := jsonq.NewQuery(data)

	price, err = jq.Float("prices", "3", "low_estimate")
	if err != nil {
		fmt.Println(err)
	}

	duration, err = jq.Float("prices", "3", "duration")
	if err != nil {
		fmt.Println(err)
	}

	distance, err = jq.Float("prices", "3", "distance")
	if err != nil {
		fmt.Println(err)
	}

	return
}

// Request a ride
func requestRide(lat1, lng1, lat2, lng2 float64, productId string) (eta float64) {
	fdata := url.Values{}
	fdata.Set("product_id", productId)
	fdata.Add("start_latitude", FloatToString(lat1))
	fdata.Add("start_longitude", FloatToString(lng1))
	fdata.Add("end_latitude", FloatToString(lat2))
	fdata.Add("end_longitude", FloatToString(lng2))
	client := &http.Client{}
	req, _ := http.NewRequest("POST", "https://sandbox-api.uber.com/v1/requests", bytes.NewBufferString(fdata.Encode()))
	req.Header.Add("Authorization", "Token 7QTVuOgroe-PWmAXhGiQUuzhVYaKR-DrdKq9uaYd")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(fdata.Encode())))
	res, _ := client.Do(req)

	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		log.Fatal(err)
	}

	if res.StatusCode != 200 {
		log.Fatal("................Unexpected status code", res.StatusCode)
	}
	data := map[string]interface{}{}
	dec := json.NewDecoder(strings.NewReader(string(body)))
	err = dec.Decode(&data)
	if err != nil {
		fmt.Println(err)
	}
	jq := jsonq.NewQuery(data)

	eta, err = jq.Float("products", "0", "product_id")
	if err != nil {
		fmt.Println(err)
	}
	return
}

type Request struct {
	Start     string   `json:"starting_from_location_id"`
	Locations []string `json:"location_ids"`
}

type Response struct {
	Id       bson.ObjectId `json:"id" bson:"_id,omitempty"`
	Status   string        `json:"status"`
	Start    string        `json:"starting_from_location_id"`
	Next     string        `json:"starting_from_location_id"`
	Best     []string      `json:"best_route_location_ids"`
	Cost     float64       `json:"total_uber_costs"`
	Duration float64       `json:"total_uber_duration"`
	Distance float64       `json:"total_distance"`
	Wait     float64       `json:"total_distance"`
}
type Location struct {
	Id         bson.ObjectId `json:"id" bson:"_id,omitempty"`
	Name       string        `json:"name"`
	Address    string        `json:"address"`
	City       string        `json:"city"`
	State      string        `json:"state"`
	Zip        string        `json:"zip"`
	Coordinate `json:"coordinate"`
}

type Coordinate struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

//Handle all requests
func Handler(response http.ResponseWriter, request *http.Request) {
	response.Header().Set("Content-type", "text/html")
	webpage, err := ioutil.ReadFile("index.html")
	if err != nil {
		http.Error(response, fmt.Sprintf("home.html file error %v", err), 500)
	}
	fmt.Fprint(response, string(webpage))
}

func APIHandler(response http.ResponseWriter, request *http.Request) {
	session, err := mgo.Dial("mongodb://sushain:1234@ds043694.mongolab.com:43694/tripdb")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	// Optional. Switch the session to a monotonic behavior.
	session.SetMode(mgo.Monotonic, true)

	tripC := session.DB("locations").C("trips")
	locationsC := session.DB("locations").C("locations")

	//set mime type to JSON
	response.Header().Set("Content-type", "application/json")

	err = request.ParseForm()
	if err != nil {
		http.Error(response, fmt.Sprintf("error parsing url %v", err), 500)
	}

	var result Response
	id := strings.Replace(request.URL.Path, "/trips/", "", -1)
	switch request.Method {
	case "GET":
		if id != "" {
			err = tripC.Find(bson.M{"_id": bson.ObjectIdHex(id)}).One(&result)
			if err != nil {
				log.Fatal(err)
			}
		}

	case "POST":
		body, err := ioutil.ReadAll(request.Body)
		if err != nil {
			panic(err)
		}
		var request Request
		err = json.Unmarshal(body, &request)
		if err != nil {
			panic(err)
		}

		// Get lat,lang using locationId
		var location Location
		var start Location
		// Get starting location lat,lng
		err = locationsC.Find(bson.M{"_id": bson.ObjectIdHex(request.Start)}).One(&start)
		if err != nil {
			log.Fatal(err)
		}

		var totalDuration, totalCost, totalDistance float64

		//Go through all locations to get data
		for _, lid := range request.Locations {
			err = locationsC.Find(bson.M{"_id": bson.ObjectIdHex(lid)}).One(&location)
			if err != nil {
				log.Fatal(err)
			}
			price, duration, distance := uberPrice(start.Coordinate.Lat, start.Coordinate.Lng, location.Coordinate.Lat, location.Coordinate.Lng)
			totalCost += price
			totalDuration += duration
			totalDistance += distance
			// Get Price Data
			start = location
		}
		result.Start = request.Start
		result.Status = "planing"
		result.Cost = totalCost
		result.Duration = totalDuration
		result.Distance = totalDistance
		result.Best = request.Locations

		//Insert the trip into Database
		err = tripC.Insert(&result)
		if err != nil {
			fmt.Println(err)
		}

	case "PUT":

		tripId := strings.Replace(strings.Replace(request.URL.Path, "/trips/", "", -1), "/request", "", -1)
		// TODO request ride
		err = tripC.Find(bson.M{"_id": bson.ObjectIdHex(tripId)}).One(&result)
		if err != nil {
			log.Fatal(err)
		}

		//Get location lat lng
		var location Location

		err = locationsC.Find(bson.M{"_id": bson.ObjectIdHex(result.Start)}).One(&location)
		if err != nil {
			log.Fatal(err)
		}

		// Get product ( uberX)
		productId := getProduct(location.Coordinate.Lat, location.Coordinate.Lng)
		fmt.Println("Got product ID:", productId)

		//Request a ride
		best := result.Best

		var next Location
		err = locationsC.Find(bson.M{"_id": bson.ObjectIdHex(best[0])}).One(&next)
		if err != nil {
			log.Fatal(err)
		}

		// Request a ride
		//eta := requestRide(location.Coordinate.Lat, location.Coordinate.Lng, next.Coordinate.Lat, next.Coordinate.Lng, productId)
		//fmt.Println("got ETA:", eta)
		//Update datbaase
		err = tripC.Update(bson.M{"_id": bson.ObjectIdHex(tripId)}, bson.M{"$set": bson.M{"status": "requesting", "next": best[0]}})
		if err != nil {
			log.Fatal(err)
		}

		err = tripC.Find(bson.M{"_id": bson.ObjectIdHex(tripId)}).One(&result)
		if err != nil {
			log.Fatal(err)
		}

	default:
	}

	json, err := json.Marshal(&result)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Send the text diagnostics to the client.
	fmt.Fprintf(response, "%v", string(json))

}
func main() {
	port := 8080
	var err string
	portstring := strconv.Itoa(port)

	mux := http.NewServeMux()
	mux.Handle("/trips/", http.HandlerFunc(APIHandler))
	mux.Handle("/", http.HandlerFunc(Handler))

	// Start listing on a given port with these routes on this server.
	log.Print("Listening on port " + portstring + " ... ")
	errs := http.ListenAndServe(":"+portstring, mux)
	if errs != nil {
		log.Fatal("ListenAndServe error: ", err)
	}
}
