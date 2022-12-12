package db

import (
	"database/sql"
	"fmt"
	// "io/ioutil"
	"log"

	_ "github.com/mattn/go-sqlite3"
	// "github.com/tidwall/gjson"
	"golang.org/x/exp/slices"
	"gopkg.in/guregu/null.v3"
)

var count = 0

type Stream interface {
	StreamItem | FavouriteItem
}

type V[T Stream] []T
	
type I struct {
	
}

type Item[T Stream] struct {
	V[T]
}

type FavouriteItem struct {
	Id sql.NullInt64
	StreamName, Logo, Url null.String
}

type StreamItem struct {
	Id sql.NullInt64 
	StreamName, Country, Logo, Url null.String
}

func NewStreamItem() StreamItem {
	return StreamItem{}
}

// type Favorites [T FavouriteItem] []FavouriteItem

func (favouriteItem FavouriteItem) ToStream() StreamItem {
	var index sql.NullInt64
	var country null.String
	index.Scan(nil)
	country.Scan("undefined")
	stream := StreamItem{Id: favouriteItem.Id, StreamName: favouriteItem.StreamName, Country: country, Logo: 
		favouriteItem.Logo, Url: favouriteItem.Url}
	return stream
}	

type StreamDatabase struct {
	*sql.DB
}

func InitDB(file string) *StreamDatabase {
	db, err := sql.Open("sqlite3", file)
	if err != nil {
		panic(err)
	}
	streamDatabase := &StreamDatabase{DB: db}
	return streamDatabase
}

// func (db *StreamDatabase) LoadItemByIndex(key string) []StreamItem {
// 	file, err := ioutil.ReadFile("db/metadata.json")
// 	if err != nil {
// 		fmt.Println(err)
// 	}
// 	var elems = gjson.GetBytes(file, key).Array()
// 	streamList := []StreamItem{}
// 	for _, elem := range elems {
// 		fmt.Println(elem.String())
// 		row := db.QueryRow("SELECT * FROM radiometadata WHERE Id = ?", elem.String())
// 		var stream = NewStreamItem()
// 		err := row.Scan(&stream.Id, &stream.StreamName, &stream.Country, &stream.Logo, &stream.Url)
// 		if err != nil {
// 			panic(err)
// 		}
// 		streamList = append(streamList, stream)
// 	}
// 	return streamList

// } 

func (db *StreamDatabase) LoadLandList() []string {
	rows, err := db.Query("SELECT Country FROM radiometadata")
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	landList := []string{}
	for rows.Next() {
		var s string
		err := rows.Scan(&s)
		if err != nil {
			fmt.Println(err)
		}
		if !slices.Contains(landList, s) {
			landList = append(landList, s)
		}
	}
	return landList
}

func (db StreamDatabase) LoadStationListFromCountry(argv interface{}) []StreamItem {
	var query = "SELECT StreamName, Country, Logo, Url FROM radiometadata WHERE Country = ?"
	rows, err := db.Query(query, argv)
	if err != nil {
		panic(err)
	}
	streamlist := db.LoadData(rows)
	return streamlist
}

// func (db *StreamDatabase) LoadStationListFromCountry_2(country string) []StreamItem {
// 	return db.LoadItemByIndex(country)
// }

func (db *StreamDatabase) LoadStationList(argv interface{}) []StreamItem {
	var query = "SELECT StreamName, Country, Logo, Url FROM radiometadata"
	var streamlist = []StreamItem{}
	if argv != nil {
		var query = query + " WHERE StreamName MATCH  ?"
		rows, err := db.Query(query, "" + fmt.Sprintf("%v", argv) + "")
		if err != nil {
			panic(err)
		}
		streamlist = db.LoadData(rows)
	} else {
		rows, err := db.Query(query)
		if err != nil {
			panic(err)
		}
		streamlist = db.LoadData(rows)
	}
	return streamlist
}

func (db *StreamDatabase) LoadData(rows *sql.Rows) []StreamItem {
	streamList := []StreamItem{}
	defer rows.Close()
	for rows.Next() {
		var StreamItem = NewStreamItem()
		err := rows.Scan(&StreamItem.StreamName, 
			&StreamItem.Country, &StreamItem.Logo, &StreamItem.Url)
		if err != nil {
			fmt.Println(err)
		}
		streamList = append(streamList, StreamItem)
	}
	return streamList
}

func (db *StreamDatabase) AddToFavourites(nameOfStream, urlOfStream, iconOfSstream string) {
	row := db.QueryRow("SELECT max(Id), StreamName FROM favourites")
	var id int64
	var streamName null.String
	row.Scan(&id, &streamName)
	if count > 0 || len(streamName.String) > 0{
		id++
	}
	
	var query = "INSERT INTO favourites VALUES (:Id, :StreamName, :Logo, :Url)"	
	_, err := db.Exec(query, id, nameOfStream, iconOfSstream, urlOfStream)
	if err != nil {
		fmt.Println(err)
	}
	count++

}

func (db *StreamDatabase) LoadFavourites() []FavouriteItem {
	var query = "SELECT StreamName, Logo, Url FROM favourites"
	rows, err := db.Query(query)
	if err != nil {
		panic(err)
	}
	favourites := []FavouriteItem{}
	defer rows.Close()
	for rows.Next() {
		var fv FavouriteItem
		err := rows.Scan(&fv.StreamName, &fv.Logo, &fv.Url)
		if err != nil {
			fmt.Println(err)
		}
		favourites = append(favourites, fv)
	}
	return favourites
}

func (db *StreamDatabase) GetFavouritesByItemName(name string) (FavouriteItem, error) {
	row := db.QueryRow("SELECT Id,  StreamName, Logo, Url FROM favourites WHERE StreamName = ?", name)
	
	var fv FavouriteItem
	err := row.Scan(&fv.Id, &fv.StreamName, &fv.Logo, &fv.Url)
	if err != nil {
		if err == sql.ErrNoRows {
			fv = FavouriteItem{}
			return fv, err
		} else {
			log.Fatal(err)
		}
	}
	return fv, nil
}

func (db *StreamDatabase) RemoveFavoriteItem(id int) {
	db.Exec("DELETE from favourites WHERE Id=?", id)
	var streamName string
	var streamNames []string
	rows, _ := db.Query("SELECT StreamName FROM favourites")
	defer rows.Close()
	for rows.Next() {
		rows.Scan(&streamName)
		streamNames = append(streamNames, streamName)
	}
	for ind, elem := range streamNames {
		_, err := db.Exec("UPDATE favourites SET Id = ? WHERE StreamName = ?", ind, elem)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func (db *StreamDatabase) GetStreamByItemName(streamName string) (StreamItem, error) {
	row := db.QueryRow("SELECT StreamName, Logo, Url FROM radiometadata WHERE StreamName = ?", streamName)
	var item = NewStreamItem()
	err := row.Scan(&item.StreamName, &item.Logo, &item.Url)
	if err != nil {
		if err == sql.ErrNoRows {
			item := NewStreamItem()
			return item, err
		} else {
			log.Fatal(err)
		}
	}
	return item, nil
}

func (db *StreamDatabase) Update(oldStreamName, newStreamName, NewUrl string) {
	fv, err := db.GetFavouritesByItemName(oldStreamName)
	id := fv.Id
	fmt.Println(fv.StreamName.String)
	if err == nil {
		db.Exec("UPDATE favourites SET StreamName = ?, Url = ? WHERE Id = ?", newStreamName, NewUrl, id)
	} else {
		log.Fatal(err)
		item, err := db.GetStreamByItemName(oldStreamName)
		id := item.Id
		if err == nil {
			db.Exec("UPDATE radiometadata SET StreamName = ?, Url = ? WHERE Id = ?", newStreamName, NewUrl, id)
		} else {
			log.Fatal(err)
		}
	}
}