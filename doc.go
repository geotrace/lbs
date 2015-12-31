// Работа с внутренней базой для определения географических координат по данным вышек сотовой
// станции.
//
// Интерфейс запросов и ответов полностью совпадает с интерфейсом
// [github.com/geotrace/locator](https://github.com/geotrace/locator/), поэтому данная библиотека
// может использоваться как замена удаленных сервисов геолокации Mozilla, Yandex или Google.
// В качестве наполнения базы данных можно использовать данные, предоставляемые OpenCellID или
// Mozilla Locator.
//
// В качестве хранилища для данных используется MongoDB.
//
// В состав библиотеке так же входит программа
// [`lbs-import`](https://github.com/geotrace/lbs/tree/master/lbs-import), для импорта данных о
// сотовых вышках и их координатах, представленных в формате CSV.
package lbs
