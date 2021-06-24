package middleware

import (
	"manago"
	"strings"

	"manago/sample"
)

type InGroupMiddleware struct {
	groupsToCheck []string
}

func NewInGroupMiddleware(groups ...string) InGroupMiddleware {
	igm := InGroupMiddleware{}
	igm.SetGroups(groups...)
	return igm
}

func (igm *InGroupMiddleware) SetGroups(groups ...string) {
	igm.groupsToCheck = groups
}

func (igm *InGroupMiddleware) RunBefore(ctr *manago.Controller) (proceed bool) {
	user := &sample.User{}
	err := ctr.AuthGetUser(user, "Groups")
	if err != nil {
		ctr.SetRedir("/user/login")
		return
	}

	for _, groupName := range igm.groupsToCheck {
		for _, group := range user.Groups {
			if strings.EqualFold(groupName, group) {
				proceed = true
				return
			}
		}
	}

	ctr.SetError(401, nil, "Brak dostępu, weryfikacja grupy użytkownika negatywna")
	return
}

func (igm *InGroupMiddleware) RunAfter(ctr *manago.Controller) {

}
