package manago

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

type ViewSet struct {
	templatesLocation string
	partialsLocation  string
	ts                map[string]*template.Template
}

func (vs *ViewSet) Load(conf *Config) (err error) {

	vs.ts = make(map[string]*template.Template)

	var layouts, partials *template.Template

	vs.templatesLocation = conf.TemplatesPath
	vs.partialsLocation = vs.templatesLocation + "partials/"

	layouts, err = vs.parseFolder(vs.templatesLocation, ".gohtml", nil)
	if err != nil {
		err = fmt.Errorf("templates NewViews: parsing base failed: %w", err)
		return
	}

	partials, err = vs.parseFolder(vs.partialsLocation, ".html", layouts)
	if err != nil {
		err = fmt.Errorf("templates NewViews parsing partials failed: %w", err)
		return
	}

	err = vs.parseTemplatesFiles(partials, vs.templatesLocation, strings.Trim(vs.partialsLocation, "."))

	if err != nil {
		err = fmt.Errorf("templates NewViews: %w", err)
	}

	return
}

func (vs *ViewSet) parseTemplatesFiles(layouts *template.Template, dirName string, dirExcluded string) error {
	log.Print("Looking for templates in: ", dirName)

	files, err := ioutil.ReadDir(dirName)
	if err != nil {
		panic("cannot read templates directory")
	}
	if !strings.Contains(dirName, dirExcluded) {
		// regular templates
		log.Printf("Found %d files in %s", len(files), dirName)
		for _, file := range files {

			filename := file.Name()

			if file.IsDir() {
				err = vs.parseTemplatesFiles(layouts, dirName+filename+"/", dirExcluded)
				if err != nil {
					return err
				}
			} else if strings.HasSuffix(filename, ".html") {
				// regular template
				shortDir := strings.TrimPrefix(dirName, vs.templatesLocation)
				name := shortDir + strings.TrimSuffix(filename, ".html")
				log.Print("Saving on: ", name, " file: ", filename, " in dir: ", dirName)

				t, cerr := layouts.Clone()
				if cerr != nil {
					return cerr
				}
				vs.ts[name], err = t.New(name).ParseFiles(dirName + filename)
				if err != nil {
					return err
				}
			}
		}
	}

	return err
}

func (vs *ViewSet) parseFolder(dir string, fileExtension string, sT *template.Template) (*template.Template, error) {
	log.Printf("Looking for template files[%v] in: %v", fileExtension, dir)
	var filesToParse []string

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		return sT, nil
	}

	for _, file := range files {

		filename := file.Name()

		if strings.HasSuffix(filename, fileExtension) {
			filesToParse = append(filesToParse, dir+filename)
		}
	}

	log.Printf("Found and parsing %d files in %v.", len(filesToParse), dir)

	if sT == nil {
		return template.New("zero").Funcs(template.FuncMap{
			"isNot":   tFuncIsNot,
			"tSimple": tTime,
			"tDate":   tDate,
			"sLimit":  tLimitString,
			"tFindInput":  tFindInput,
		}).ParseFiles(filesToParse...)
	} else {
		return sT.ParseFiles(filesToParse...)
	}

}

func (vs *ViewSet) GetT(name string) *template.Template {
	t, ok := vs.ts[name]
	if ok {
		return t
	} else {
		log.Print("template not present in the ts map, using default")
		tDef, present := vs.ts["default"]
		if !present {
			log.Fatal("ViewSet GetT: default template not found! Did you add 'default.html'?")
			return nil
		}
		return tDef
	}

}

func (vs *ViewSet) FireTemplate(name string, w http.ResponseWriter, ctnt *map[string]interface{}) error {
	return vs.GetT(name).ExecuteTemplate(w, "base.gohtml", *ctnt)
}

func tFuncIsNot(val interface{}) (ret bool) {
	ret = false

	switch val := val.(type) {
	case bool:
		ret = !val
	case int:
		ret = val == 0
	case string:
		ret = len(val) == 0

	}

	return
}

func tTime(t time.Time) string {
	return t.Format("15:04 2006-01-02")
}

func tDate(t time.Time) string {
	return t.Format("2006-01-02")
}

func tLimitString(s string) string {
	if len(s) > 18 {
		return s[:16] + "..."
	}
	return s
}

func tFindInput(s ...string) (output map[string]string) {
	output = make(map[string]string)

	if len(s) < 3 {
		return 
	}
	
	output["Title"] = s[0]
	output["ModelName"] = s[1]
	output["FindPost"] = s[2]
	
	if len(s) < 4 {
		return 
	}

	output["FindFields"] = s[3]

	return
}
