package manago

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/jinzhu/gorm"
)

type Paginator struct {
	CurrentPage int
	Limit       int

	LastPage    int
	Count       int
	IsFirstPage bool
	IsLastPage  bool

	controller *Controller
}

func NewPaginator(ctr *Controller) (pag *Paginator) {
	pag = &Paginator{
		Limit:       15,
		controller:  ctr,
		CurrentPage: ctr.Req.ParamIntByName("page"),
	}

	return
}

func (pag *Paginator) RunTransaction(modelSlice interface{}, db *gorm.DB) error {
	kind := reflect.ValueOf(modelSlice).Type().Kind().String()
	if kind != "ptr" {
		return fmt.Errorf("Paginator RunTransaction: Expected pointer to modele slice! Received non-pointer type.")
	}

	var count int64

	db.Find(modelSlice).Count(&count)

	err := db.Limit(pag.getLimit()).Offset(pag.getOffset()).Order("id").Find(modelSlice).Error
	if err != nil {
		return fmt.Errorf("Paginator RunTransaction: gorm DB.Find error:\n%v", err)
	}

	pag.updateCount(count)

	return nil
}

func (pag *Paginator) QueryInFields(modelSlice interface{}, query string, fields ...string) error {

	tx := pag.controller.Man.Dbc.DB
	for ix, field := range fields {
		if ix == 0 {
			tx = tx.Where(fmt.Sprintf("LOWER(%s) LIKE ?", field), "%"+strings.ToLower(query)+"%")
		} else {
			tx = tx.Or(fmt.Sprintf("LOWER(%s) LIKE ?", field), "%"+strings.ToLower(query)+"%")
		}
	}

	return pag.RunTransaction(modelSlice, tx)
}

func (pag *Paginator) MsSqlOffsetLine() string {

	return fmt.Sprintf("OFFSET %d ROWS FETCH NEXT %d ROWS ONLY", pag.getOffset(), pag.getLimit())

}

func (pag *Paginator) FindModel(modelSlice interface{}, query interface{}) error {
	return pag.RunTransaction(modelSlice, pag.controller.Man.Dbc.DB.Where(query))
}

func (pag *Paginator) HasRelation(modelSlice interface{}, relationName string) error {
	tx := pag.controller.Man.Dbc.DB.Where(strings.ToLower(relationName) + "_id > 0")
	return pag.RunTransaction(modelSlice, tx)
}

func (pag *Paginator) updateCount(count int64) {
	pag.Count = int(count)

	totalPages := int(count) / pag.Limit

	if totalPages*pag.Limit < pag.Count {
		totalPages += 1
	}
	pag.LastPage = totalPages - 1

	pag.IsFirstPage = (pag.CurrentPage == 0)
	pag.IsLastPage = (pag.CurrentPage == pag.LastPage)

}

func (pag *Paginator) getOffset() int {
	return pag.CurrentPage * pag.Limit
}

func (pag *Paginator) getLimit() int {
	return pag.Limit
}

func (pag *Paginator) SetPage(page int) {
	if page < pag.LastPage || pag.LastPage == 0 {
		pag.CurrentPage = page
	} else {
		pag.CurrentPage = pag.LastPage
	}
}

func (pag *Paginator) SetLimit(limit int) {
	pag.Limit = limit
}

func (pag *Paginator) GetNextPageNumber() int {

	if pag.IsLastPage {
		return pag.CurrentPage
	}

	return pag.CurrentPage + 1
}

func (pag *Paginator) GetPrevPageNumber() int {

	if pag.IsFirstPage {
		return pag.CurrentPage
	}

	return pag.CurrentPage - 1
}
