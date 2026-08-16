package publish

import internalpublish "github.com/mudongliang/weoa-cli/internal/publish"

type articleIdentity struct {
	appMsgID int64
	url      string
}

func identityOf(a internalpublish.Article) articleIdentity {
	return articleIdentity{appMsgID: a.AppMsgID, url: a.URL}
}
