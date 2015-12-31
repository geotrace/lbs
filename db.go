// Работа с внутренней базой для определения географических координат по данным вышек сотовой
// станции.
//
// Интерфейс запросов и ответов полностью совпадает с интерфейсом github.com/geotrace/locator,
// поэтому данная библиотека может использоваться как замена удаленных сервисов геолокации Mozilla,
// Yandex или Google.
//
// В качестве наполнения базы данных можно использовать данные, предоставляемые OpenCellID или
// Mozilla Locator.
//
// В качестве хранилища для данных используется MongoDB.
//
// В состав библиотеке так же входит программа lbs-import, для импорта данных о сотовых вышках и их
// координатах, представленных в формате CSV.
package lbs

import (
	"errors"
	"math"

	"github.com/geotrace/geo"
	"github.com/geotrace/locator"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var CollectionName = "lbs"   // описывает название коллекции с данными для LBS.
var DefaultRadioType = "gsm" // используемый по умолчанию тип радио.

// DB описывает хранилище LBS данных и работу с ними.
type DB struct {
	name    string       // название базы данных
	session *mgo.Session // хранилище MogoDB
}

// InitDB возвращает инициализированный объект для работы с хранилищем LBS данных.
func InitDB(session *mgo.Session, name string) (db *DB, err error) {
	db = &DB{
		session: session,
		name:    name,
	}
	return
}

// Key описывает ключ для поиска информации по LBS.
type Key struct {
	RadioType         string `bson:"radio"` // The mobile radio type. Supported values are lte, gsm, umts, cdma, and wcdma.
	MobileCountryCode uint16 `bson:"mcc"`   // country code  (250 - Россия, 255 - Украина, Беларусь - 257)
	MobileNetworkCode uint16 `bson:"mnc"`   // operator code
	LocationAreaCode  uint16 `bson:"lac"`   // the base station cell number
	CellId            uint32 `bson:"cell"`  // base station number
}

// Data описывает данные для вышки сотовой станции.
type Data struct {
	Location geo.Point `bson:"location"` // координаты
	Accuracy float64   `bson:"range"`    // расстояние
}

var (
	ErrEmptyRequest = errors.New("lbs: empty request")
	ErrNotFound     = errors.New("lbs: not found")
)

// GetCells возвращает информацию о найденных сотовых станциях.
func (db *DB) GetCells(req locator.Request) (cells []Data, err error) {
	if len(req.CellTowers) == 0 && len(req.WifiAccessPoints) == 0 {
		return nil, ErrEmptyRequest
	}
	radio, mcc, mnc := req.RadioType, req.HomeMobileCountryCode, req.HomeMobileNetworkCode
	if radio == "" {
		radio = DefaultRadioType
	}
	if mcc == 0 {
		mcc = req.CellTowers[0].MobileCountryCode
	}
	if mnc == 0 {
		mnc = req.CellTowers[0].MobileNetworkCode
	}
	// формируем запрос на получение данных о всех вышках
	cellsData := make([]bson.M, len(req.CellTowers))
	for i, cell := range req.CellTowers {
		cellsData[i] = bson.M{
			"lac":  cell.LocationAreaCode,
			"cell": cell.CellId,
		}
	}
	search := bson.M{
		"radio": radio,
		"mcc":   mcc,
		"mnc":   mnc,
		"$or":   cellsData,
	}
	// фильтруем поля получаемых данных
	selector := bson.M{"location": 1, "range": 1, "_id": 0}
	// инициализируем приемник данных
	cells = make([]Data, 0, len(req.CellTowers))
	// запрашиваем данные из коллекции
	session := db.session.Copy()
	coll := session.DB(db.name).C(CollectionName)
	err = coll.Find(search).Select(selector).All(&cells)
	session.Close()
	return cells, err
}

// AveragePoint ищет и вычисляет координаты, переданные в запросе, на основании данных вышек сотовой
// связи. Если данных не достаточно или необходимая для вычислений информация не найдена в
// хранилище, то возвращается ошибка.
func (db *DB) Get(req locator.Request) (response *locator.Response, err error) {
	cells, err := db.GetCells(req)
	if err != nil {
		return nil, err
	}
	// перебираем полученные данные
	var lon, lat float64
	for _, cell := range cells {
		lon += cell.Location.Longitude()
		lat += cell.Location.Latitude()
	}
	count := float64(len(cells))
	lon, lat = lon/count, lat/count // вычисляем среднее значение
	const EARTH_RADIUS = 6378137.0
	var accuracy float64
	for _, cell := range cells {
		lat2 := cell.Location.Latitude()
		lon2 := cell.Location.Longitude()
		dLat := math.Pi / 180.0 * (lat2 - lat) / 2.0
		dLon := math.Pi / 180.0 * (lon2 - lon) / 2.0
		lat1 := math.Pi / 180.0 * (lat)
		lat2 = math.Pi / 180.0 * (lat2)
		a := math.Pow(math.Sin(dLat), 2) + math.Cos(lat1)*math.Cos(lat2)*math.Pow(math.Sin(dLon), 2)
		c := math.Asin(math.Min(1, math.Sqrt(a)))
		dist := 2*EARTH_RADIUS*c + cell.Accuracy
		if dist > accuracy {
			accuracy = dist
		}
	}
	response = &locator.Response{
		Location: locator.Point{
			Lat: lat,
			Lng: lon,
		},
		Accuracy: accuracy,
	}
	return response, nil
}

// Records возвращает количество записей в хранилище LBS.
func (db *DB) Records() int {
	session := db.session.Copy()
	coll := session.DB(db.name).C(CollectionName)
	total, _ := coll.Count()
	session.Close()
	return total
}
