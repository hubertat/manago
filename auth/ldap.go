package auth

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-ldap/ldap"
)

type User struct {
	Name        string
	DisplayName string
	Email       string

	Guid string

	Groups []string
}

type ErrorLoginFailed struct {
	error
}

func (e ErrorLoginFailed) Error() string {
	return "login failed, login and/or password invalid"
}

type Ldap struct {
	config Config
}

func (ld *Ldap) getUser(username string, lConn *ldap.Conn) (err error, user User) {

	searchRequest := ldap.NewSearchRequest(
		ld.config.BaseDn,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf("(samaccountname=%s)", username),
		[]string{"memberOf", "cn", "mail", "objectGUID"},
		nil,
	)

	sr, err := lConn.Search(searchRequest)
	if err != nil {
		err = errors.Join(err, errors.New("ldap search failed"))
		return
	}

	if len(sr.Entries) != 1 {
		err = errors.New("ldap search returned wrong number of entries")
		return
	}

	ent := make(map[string][]string)
	for _, attr := range sr.Entries[0].Attributes {
		ent[attr.Name] = attr.Values
	}

	if len(ent["objectGUID"]) != 1 {
		err = errors.New("ldap search returned wrong number of GUIDs")
		return
	}

	user.Name = username
	user.Guid = ent["objectGUID"][0]
	user.DisplayName = ent["cn"][0]

	for _, grcn := range ent["memberOf"] {
		longName := strings.Split(grcn, ",")
		if len(longName) > 0 {
			grName := strings.Split(longName[0], "=")
			if len(grName) > 1 {
				user.Groups = append(user.Groups, grName[1])
			}
		}

	}

	return
}

func (ld *Ldap) Authenticate(username, password string) (err error, guid string) {
	curatedUsername := strings.Split(username, "@")[0]
	splitSlash := strings.Split(curatedUsername, `\`)
	if len(splitSlash) > 1 {
		curatedUsername = splitSlash[1]
	}

	l, err := ldap.Dial("tcp", fmt.Sprintf("%s:%d", ld.config.Host, ld.config.Port))
	if err != nil {
		err = errors.Join(err, errors.New("ldap dial failed"))
		return
	}

	defer l.Close()

	err = l.Bind(fmt.Sprintf("%s@%s", curatedUsername, ld.config.Domain), password)
	if err != nil {
		err = errors.Join(err, ErrorLoginFailed{})
		return
	}

	return
}

func NewLdap(config Config) (ldap *Ldap, err error) {
	ldap = &Ldap{
		config: config,
	}
	return
}
