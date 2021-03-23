package manago

import (
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"time"
	// "embed"
)

type ViewSet struct {
	templatesLocation string
	partialsLocation  string
	ts                map[string]*template.Template
	baseTemplate	*template.Template
	man				*Manager			
}


// var staticEmbeded embed.FS

func (vs *ViewSet) Load(conf *Config, manager *Manager) (err error) {
	vs.man = manager
	vs.ts = make(map[string]*template.Template)

	if manager.StaticFsys == nil {
		err = fmt.Errorf("No static FS! Aborting")
		return
	}

	vs.templatesLocation = strings.Trim(conf.TemplatesPath, "\\/.")
	vs.partialsLocation = "/partials"

	

	err = fs.WalkDir(manager.StaticFsys, vs.templatesLocation, vs.walkForBase)
	if err != nil {
		err = fmt.Errorf("templates Load: walking for base failed: %w", err)
	}

	err = fs.WalkDir(manager.StaticFsys, vs.templatesLocation, vs.walkFolders)

	if err != nil {
		err = fmt.Errorf("templates Load: walking dir failed: %w", err)
	}

	return
}

func (vs *ViewSet) walkFolders(path string, d fs.DirEntry, err error) error {
	if err != nil {
		return fmt.Errorf("walkFolders received error: %v", err)
	}

	if !strings.Contains(path, vs.partialsLocation) && strings.Contains(strings.ToLower(d.Name()), ".html") && !d.IsDir() {
		// .html file, treating like template
		name := strings.TrimPrefix(strings.TrimSuffix(path, ".html"), vs.templatesLocation)
		name = strings.Trim(name, "\\/.")
		log.Printf("walkFolders found template file: %s and saving as: %s", d.Name(), name)

		
		t, tempErr := vs.baseTemplate.Clone()
		if tempErr != nil {
			return fmt.Errorf("walkFolders error when cloning template: %v\n", tempErr)
		}
		vs.ts[name], tempErr = t.New(name).ParseFS(vs.man.StaticFsys, path)
		if err != nil {
			return fmt.Errorf("walkFolders error when template ParseFiles: %v\n", tempErr)
		}
	}

	return nil
}

func (vs *ViewSet) walkForBase(path string, d fs.DirEntry, err error) error {
	if err != nil {
		return fmt.Errorf("walkForBase received error: %v", err)
	}

	if !d.IsDir() {
		if strings.Contains(strings.ToLower(d.Name()), ".gohtml") || (strings.Contains(path, vs.partialsLocation) && strings.Contains(strings.ToLower(d.Name()), ".html") ) {
			name := strings.ToLower(d.Name())
			log.Printf("Found and parsing %s as %s", path, name)

			var tempErr error

			if vs.baseTemplate == nil {
				vs.baseTemplate, tempErr = template.New("zero").Funcs(template.FuncMap{
					"isNot":   tFuncIsNot,
					"tSimple": tTime,
					"tDate":   tDate,
					"sLimit":  tLimitString,
					"tFindInput":  tFindInput,
					"uintToString": uintToString,
				}).ParseFS(vs.man.StaticFsys, path)
			} else {
				vs.baseTemplate, tempErr = vs.baseTemplate.ParseFS(vs.man.StaticFsys, path)
			}
			if tempErr != nil {
				return fmt.Errorf("walkForBase error from parsing template: %v\n", tempErr)
			}
		}
	}
	
	return nil
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

	if len(s) < 6 {
		return
	}

	output["SelectedOption"] = "true"
	output["SelectedVal"] = s[4]
	output["SelectedName"] = s[5]
	
	return
}

func uintToString(val uint) string {
	return fmt.Sprintf("%d", val)
}