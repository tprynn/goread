/*
 * Copyright (c) 2013 Matt Jibson <matt.jibson@gmail.com>
 *
 * Permission to use, copy, modify, and distribute this software for any
 * purpose with or without fee is hereby granted, provided that the above
 * copyright notice and this permission notice appear in all copies.
 *
 * THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
 * WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
 * MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
 * ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
 * WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
 * ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
 * OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
 */

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	// "github.com/tprynn/goread/_third_party/github.com/MiniProfiler/go/miniprofiler"
	
	"github.com/tprynn/goread/_third_party/github.com/gorilla/mux"
	"github.com/mjibson/goon"

	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
)

var (
	router      = new(mux.Router)
	templates   *template.Template
	mobileIndex []byte
)

func init() {
	var err error
	if templates, err = template.New("").Funcs(funcs).
		ParseFiles(
			"templates/base.html",
			"templates/admin-all-feeds.html",
			"templates/admin-date-formats.html",
			"templates/admin-feed.html",
			"templates/admin-stats.html",
			"templates/admin-user.html",
		); err != nil {
		// log.Criticalf(appengine.NewContext(), "Failed to parse - %v", err)
		panic(err);
	}
	mobileIndex, err = ioutil.ReadFile("static/index.html")
	if err != nil {
		// log.Criticalf(appengine.NewContext(), "Couldn't read index.html - %v", err)
		panic(err);
	}

	// miniprofiler.ToggleShortcut = "Alt+C"
	// miniprofiler.Position = "bottomleft"
}

func RegisterHandlers(r *mux.Router) {
	router = r
	router.Handle("/", wrap(Main)).Name("main")
	router.Handle("/login/google", wrap(LoginGoogle)).Name("login-google")
	router.Handle("/login/redirect", wrap(LoginRedirect))
	router.Handle("/logout", wrap(Logout)).Name("logout")
	router.Handle("/push", wrap(SubscribeCallback)).Name("subscribe-callback")
	router.Handle("/tasks/import-opml", wrap(ImportOpmlTask)).Name("import-opml-task")
	router.Handle("/tasks/subscribe-feed", wrap(SubscribeFeed)).Name("subscribe-feed")
	router.Handle("/tasks/update-feed-last", wrap(UpdateFeedLast)).Name("update-feed-last")
	router.Handle("/tasks/update-feed-manual", wrap(UpdateFeed)).Name("update-feed-manual")
	router.Handle("/tasks/update-feed", wrap(UpdateFeed)).Name("update-feed")
	router.Handle("/tasks/update-feeds", wrap(UpdateFeeds)).Name("update-feeds")
	router.Handle("/tasks/delete-old-feeds", wrap(DeleteOldFeeds)).Name("delete-old-feeds")
	router.Handle("/tasks/delete-old-feed", wrap(DeleteOldFeed)).Name("delete-old-feed")

	router.Handle("/user/add-subscription", wrap(AddSubscription)).Name("add-subscription")
	router.Handle("/user/delete-account", wrap(DeleteAccount)).Name("delete-account")
	router.Handle("/user/export-opml", wrap(ExportOpml)).Name("export-opml")
	router.Handle("/user/feed-history", wrap(FeedHistory)).Name("feed-history")
	router.Handle("/user/get-contents", wrap(GetContents)).Name("get-contents")
	router.Handle("/user/get-feed", wrap(GetFeed)).Name("get-feed")
	router.Handle("/user/get-stars", wrap(GetStars)).Name("get-stars")
	router.Handle("/user/import/get-url", wrap(UploadUrl)).Name("upload-url")
	router.Handle("/user/import/opml", wrap(ImportOpml)).Name("import-opml")
	router.Handle("/user/list-feeds", wrap(ListFeeds)).Name("list-feeds")
	router.Handle("/user/mark-read", wrap(MarkRead)).Name("mark-read")
	router.Handle("/user/mark-unread", wrap(MarkUnread)).Name("mark-unread")
	router.Handle("/user/save-options", wrap(SaveOptions)).Name("save-options")
	router.Handle("/user/set-star", wrap(SetStar)).Name("set-star")
	router.Handle("/user/upload-opml", wrap(UploadOpml)).Name("upload-opml")

	router.Handle("/admin/all-feeds", wrap(AllFeeds)).Name("all-feeds")
	router.Handle("/admin/all-feeds-opml", wrap(AllFeedsOpml)).Name("all-feeds-opml")
	router.Handle("/admin/user", wrap(AdminUser)).Name("admin-user")
	router.Handle("/date-formats", wrap(AdminDateFormats)).Name("admin-date-formats")
	router.Handle("/admin/feed", wrap(AdminFeed)).Name("admin-feed")
	router.Handle("/admin/subhub", wrap(AdminSubHub)).Name("admin-subhub-feed")
	router.Handle("/admin/stats", wrap(AdminStats)).Name("admin-stats")
	router.Handle("/admin/update-feed", wrap(AdminUpdateFeed)).Name("admin-update-feed")
	router.Handle("/user/charge", wrap(Charge)).Name("charge")
	router.Handle("/user/account", wrap(Account)).Name("account")
	router.Handle("/user/uncheckout", wrap(Uncheckout)).Name("uncheckout")

	//router.Handle("/tasks/delete-blobs", wrap(DeleteBlobs)).Name("delete-blobs")

	if len(PUBSUBHUBBUB_HOST) > 0 {
		u := url.URL{
			Scheme:   "http",
			Host:     PUBSUBHUBBUB_HOST,
			Path:     routeUrl("add-subscription"),
			RawQuery: url.Values{"url": {"{url}"}}.Encode(),
		}
		subURL = u.String()
	}

	if !isDevServer {
		return
	}
	router.Handle("/user/clear-feeds", wrap(ClearFeeds)).Name("clear-feeds")
	router.Handle("/user/clear-read", wrap(ClearRead)).Name("clear-read")
	router.Handle("/test/atom.xml", wrap(TestAtom)).Name("test-atom")
}

func wrap(f func(context.Context, http.ResponseWriter, *http.Request)) http.Handler {
	// handler := wrap(f)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isDevServer {
			w.Header().Add("Access-Control-Allow-Origin", r.Header.Get("Origin"))
			w.Header().Add("Access-Control-Allow-Credentials", "true")
		}
		// handler.ServeHTTP(w, r)
		c := appengine.NewContext(r);
		f(c, w, r);
	})
}

func main() {
	appengine.Main();
}

func Main(c context.Context, w http.ResponseWriter, r *http.Request) {
	ua := r.Header.Get("User-Agent")
	mobile := strings.Contains(ua, "Mobi")
	if desktop, _ := r.Cookie("goread-desktop"); desktop != nil {
		switch desktop.Value {
		case "desktop":
			mobile = false
		case "mobile":
			mobile = true
		}
	}
	if mobile {
		w.Write(mobileIndex)
	} else {
		if err := templates.ExecuteTemplate(w, "base.html", includes(c, w, r)); err != nil {
			log.Errorf(c, "%v", err)
			serveError(w, err)
		}
	}
}

func addFeed(c context.Context, userid string, outline *OpmlOutline) error {
	gn := goon.FromContext(c)
	o := outline.Outline[0]
	log.Infof(c, "adding feed %v to user %s", o.XmlUrl, userid)
	fu, ferr := url.Parse(o.XmlUrl)
	if ferr != nil {
		return ferr
	}
	fu.Fragment = ""
	o.XmlUrl = fu.String()

	f := Feed{Url: o.XmlUrl}
	if err := gn.Get(&f); err == datastore.ErrNoSuchEntity {
		if feed, stories, err := fetchFeed(c, o.XmlUrl, o.XmlUrl); err != nil {
			return fmt.Errorf("could not add feed %s: %v", o.XmlUrl, err)
		} else {
			f = *feed
			f.Updated = time.Time{}
			f.Checked = f.Updated
			f.NextUpdate = f.Updated
			f.LastViewed = time.Now()
			gn.Put(&f)
			for _, s := range stories {
				s.Created = s.Published
			}
			if err := updateFeed(c, f.Url, feed, stories, false, false, false); err != nil {
				return err
			}

			o.XmlUrl = feed.Url
			o.HtmlUrl = feed.Link
			if o.Title == "" {
				o.Title = feed.Title
			}
		}
	} else if err != nil {
		return err
	} else {
		o.HtmlUrl = f.Link
		if o.Title == "" {
			o.Title = f.Title
		}
	}
	o.Text = ""

	return nil
}

func mergeUserOpml(c context.Context, ud *UserData, outlines ...*OpmlOutline) error {
	var fs Opml
	json.Unmarshal(ud.Opml, &fs)
	urls := make(map[string]bool)

	for _, o := range fs.Outline {
		if o.XmlUrl != "" {
			urls[o.XmlUrl] = true
		} else {
			for _, so := range o.Outline {
				urls[so.XmlUrl] = true
			}
		}
	}

	mergeOutline := func(label string, outline *OpmlOutline) {
		if _, present := urls[outline.XmlUrl]; present {
			return
		} else {
			urls[outline.XmlUrl] = true

			if label == "" {
				fs.Outline = append(fs.Outline, outline)
			} else {
				done := false
				for _, ol := range fs.Outline {
					if ol.Title == label && ol.XmlUrl == "" {
						ol.Outline = append(ol.Outline, outline)
						done = true
						break
					}
				}
				if !done {
					fs.Outline = append(fs.Outline, &OpmlOutline{
						Title:   label,
						Outline: []*OpmlOutline{outline},
					})
				}
			}
		}
	}

	for _, outline := range outlines {
		if outline.XmlUrl != "" {
			mergeOutline("", outline)
		} else {
			for _, o := range outline.Outline {
				mergeOutline(outline.Title, o)
			}
		}
	}

	b, err := json.Marshal(&fs)
	if err != nil {
		return err
	}
	ud.Opml = b
	return nil
}
