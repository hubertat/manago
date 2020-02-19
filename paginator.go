package manago

import (
	"fmt"
	"reflect"
	"strings"
)

type Paginator struct {
	CurrentPage		int
	Limit			int

	LastPage		int
	Count			int
	IsFirstPage		bool
	IsLastPage		bool


	manager			*Manager
}

func NewPaginator(man *Manager) (pag *Paginator) {
	pag = &Paginator{
		Limit: 15,
		manager: man,
	}

	return
}

func (pag *Paginator) QueryInFields(modelSlice interface{}, query string, fields ...string) error {
	kind := reflect.ValueOf(modelSlice).Type().Kind().String()
	if kind != "ptr" {
		return fmt.Errorf("Paginator QueryInFields: Expected pointer to modele slice! Received non-pointer type.")
	}

	count := 0
	tx := pag.manager.Dbc.DB
	for ix, field := range fields {
		if ix == 0 {
			tx = tx.Where(fmt.Sprintf("LOWER(%s) LIKE ?", field), "%" + strings.ToLower(query) + "%")	
		} else {
			tx = tx.Or(fmt.Sprintf("LOWER(%s) LIKE ?", field), "%" + strings.ToLower(query) + "%")
		}
	}
	
	tx.Find(modelSlice).Count(&count)
	
	err := tx.Limit(pag.getLimit()).Offset(pag.getOffset()).Find(modelSlice).Error
	if err != nil {
		return fmt.Errorf("Paginator QueryInFields: gorm DB.Find error:\n%v", err)
	}

	pag.updateCount(count)

	return nil
}

func (pag *Paginator) FindModel(modelSlice interface{}, query interface{}) error {
	kind := reflect.ValueOf(modelSlice).Type().Kind().String()
	if kind != "ptr" {
		return fmt.Errorf("Paginator FindModel: Expected pointer to modele slice! Received non-pointer type.")
	}

	count := 0
	pag.manager.Dbc.DB.Where(query).Find(modelSlice).Count(&count)
	
	err := pag.manager.Dbc.DB.Where(query).Limit(pag.getLimit()).Offset(pag.getOffset()).Find(modelSlice).Error
	if err != nil {
		return fmt.Errorf("Paginator FindModel: gorm DB.Find error:\n%v", err)
	}

	pag.updateCount(count)

	return nil
}

func (pag *Paginator) updateCount(count int) {
	pag.Count = count

	totalPages := count / pag.Limit

	if totalPages * pag.Limit < pag.Count {
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