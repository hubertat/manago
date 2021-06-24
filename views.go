package manago

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"
)

type ViewSet struct {
	man *Manager
}

func (vs *ViewSet) Load(conf *Config, manager *Manager) (err error) {
	vs.man = manager

	if manager.StaticFsys == nil {
		err = fmt.Errorf("no static FS! Aborting")
		return
	}

	return
}

func (vs *ViewSet) FireTemplate(name string, w http.ResponseWriter, ctnt *map[string]interface{}) (err error) {
	tpl, err := template.ParseFS(vs.man.StaticFsys, strings.Join([]string{"templates", name}, "/"))
	if err != nil {
		return
	}
	return tpl.Funcs(template.FuncMap{
		"isNot":        tFuncIsNot,
		"tSimple":      tTime,
		"tDate":        tDate,
		"sLimit":       tLimitString,
		"tFindInput":   tFindInput,
		"uintToString": uintToString,
		"sLimitVar":    tLimitStringVar,
		"extractHrefs": ExtractHrefs,
		"tSince":       tPrettySince,
	}).ExecuteTemplate(w, "base.gohtml", *ctnt)

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

func tLimitStringVar(s string, length int) string {
	if length == 0 {
		length = 18
	}

	if len(s) > length {
		return s[:length] + "..."
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
func ExtractHrefs(input string) (hrefs []string) {
	prefixes := []string{"https://", "http://"}
	endPosition := 0
	delimeters := ` "';
	`

	for _, prefix := range prefixes {
		description := input

		for where := strings.Index(strings.ToLower(description), prefix); where > -1; where = strings.Index(strings.ToLower(description), prefix) {
			endPosition = strings.IndexAny(description[where:], delimeters)
			if endPosition > 0 {
				hrefs = append(hrefs, description[where:where+endPosition])
				description = description[where+endPosition:]
			} else {
				hrefs = append(hrefs, description[where:])
				description = ""
			}

		}
	}

	return
}

func tPrettySince(from time.Time) string {
	duration := time.Since(from)
	if duration < time.Hour {
		return fmt.Sprintf("%d minut temu", int(duration.Minutes()))
	}

	if duration < 24*time.Hour {
		return fmt.Sprintf("%.1f godzin temu", duration.Hours())
	}

	return fmt.Sprintf("%d dni temu", int(duration.Hours()/24))
}
