package auth

import (
	"fmt"
	"log"
	"strings"

	"github.com/go-ldap/ldap"
)

func (ctr *User) Login() {

	ldapHost := ctr.Man.Config.AppVariables["ldap_host"]
	ldapDomain := ctr.Man.Config.AppVariables["ldap_domain"]

	bindusername := strings.Split(ctr.Req.FormSingle("login"), "@")[0]
	splitSlash := strings.Split(bindusername, `\`)
	if len(splitSlash) > 1 {
		bindusername = splitSlash[1]
	}
	bindpassword := ctr.Req.FormSingle("password")

	log.Printf("Trying login with %v as login", bindusername)

	l, err := ldap.Dial("tcp", fmt.Sprintf("%s:%d", ldapHost, 389))
	if err != nil {
		ctr.SetError(500, fmt.Errorf("ldap dial failed:\n%v\n", err))
		log.Print(err)
		return
	}
	defer l.Close()

	err = l.Bind(fmt.Sprintf("%s@%s", bindusername, ldapDomain), bindpassword)
	if err != nil {
		// ctr.SetError(401, fmt.Errorf("User Login failed:\n%v\n", err))
		ctr.Req.SetRedir("/user/login/invalid")
		log.Print(err)
		return
	}

	searchRequest := ldap.NewSearchRequest(
		"OU=PassageCL,DC=passage,DC=cl",
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf("(samaccountname=%s)", bindusername),
		[]string{"memberOf", "cn", "mail", "objectGUID"},
		nil,
	)

	sr, err := l.Search(searchRequest)
	if err != nil {
		ctr.SetError(500, fmt.Errorf("user Login failed:\n%v", err))
		log.Print(err)
		return
	}

	log.Printf("Found %d entry", len(sr.Entries))

	if len(sr.Entries) != 1 {
		ctr.SetError(500, fmt.Errorf("user Login failed:\nLdap wrong entries count"))
		log.Print("Ldap wrong entries count")
		return
	}

	ent := make(map[string][]string)

	for _, attr := range sr.Entries[0].Attributes {
		ent[attr.Name] = attr.Values
	}

	if len(ent["objectGUID"]) < 1 {
		err = fmt.Errorf("objectGUID field missing in ldap entries. Will not proceed")
		ctr.SetError(500, err)
		log.Print(err)
		return
	}

	guid := ent["objectGUID"][0]

	user := &models.User{}

	err = ctr.Db.Where(models.User{ID: guid}).FirstOrInit(user).Error
	if err != nil {
		ctr.SetError(500, err)
		log.Printf("Error when creating new user: %v", err)
		return
	}

	user.DisplayName = ent["cn"][0]
	user.Name = bindusername

	err = ctr.Db.Save(user).Error
	if err != nil {
		ctr.SetError(500, err)
		log.Printf("Error when creating new user: %v", err)
		return
	}

	ctr.Session.Set("auth", guid)
	ctr.Session.Set("username", bindusername)

	var gNames []string
	var groups []*models.Group

	for _, grcn := range ent["memberOf"] {
		longName := strings.Split(grcn, ",")
		if len(longName) > 0 {
			grName := strings.Split(longName[0], "=")
			if len(grName) > 1 {
				gNames = append(gNames, grName[1])

				group := &models.Group{}
				ctr.Db.Where(models.Group{Name: grName[1]}).FirstOrInit(group)
				ctr.Db.Save(group)
				groups = append(groups, group)
			}
		}

	}

	ctr.Db.Model(user).Association("Groups").Replace(groups)

	ctr.SetCt("name", ent["cn"][0])
	ctr.SetCt("groups", gNames)
}
