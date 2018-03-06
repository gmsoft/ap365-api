  package main

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"

     "goji.io"
    "goji.io/pat"
    "gopkg.in/mgo.v2"
    "gopkg.in/mgo.v2/bson"
)

const database string = "base"
const collection string = "cli_106"

func ErrorWithJSON(w http.ResponseWriter, message string, code int) {  
    w.Header().Set("Content-Type", "application/json; charset=utf-8")
    w.WriteHeader(code)
    fmt.Fprintf(w, "{message: %q}", message)
}

func ResponseWithJSON(w http.ResponseWriter, json []byte, code int) {  
    w.Header().Set("Content-Type", "application/json; charset=utf-8")
    w.WriteHeader(code)
    w.Write(json)
}

type Article struct {
    Articulo    string `json:"articulo"`
    Descrip     string `json:"descrip"`
    Original    string `json:"original"`
    Precio      string `json:"precio"`
    Foto        string `json:"foto"`
    Comentario  string `json:"comentario"`
    Marca       string `json:"marca"`
    Modelos     string `json:"modelos"`
    Motores     string `json:"motores"`
}

func main() {

    //MONGO DB
    //session, err := mgo.Dial("localhost")
    session, err := mgo.Dial("192.168.10.201")
    if err != nil {
        panic(err)
    }
    defer session.Close()
	session.SetMode(mgo.Monotonic, true)
	ensureIndex(session)

    //Endpoitns API
    mux := goji.NewMux()
    //Articulos
    mux.HandleFunc(pat.Get("/articles"), allArticles(session))
    mux.HandleFunc(pat.Get("/articles/:articulo"), articleByCodigo(session))
	mux.HandleFunc(pat.Get("/articles/rubro/:rubro"), articlesByRubro(session))
	mux.HandleFunc(pat.Post("/articles"), addArticle(session))
    mux.HandleFunc(pat.Put("/articles/:articulo"), updateArticle(session))
    mux.HandleFunc(pat.Delete("/articles/:articulo"), deleteArticle(session))
    //Web Server
    http.ListenAndServe("localhost:8099", mux)
}

func ensureIndex(s *mgo.Session) {
    session := s.Copy()
    defer session.Close()

    c := session.DB(database).C(collection)

    index := mgo.Index{
        Key:        []string{"articulo"},
        Unique:     true,
        DropDups:   true,
        Background: true,
        Sparse:     true,
    }
    err := c.EnsureIndex(index)
    if err != nil {
        panic(err)
    }
}

func allArticles(s *mgo.Session) func(w http.ResponseWriter, r *http.Request) {  
    return func(w http.ResponseWriter, r *http.Request) {
        session := s.Copy()
        defer session.Close()

        c := session.DB(database).C(collection)

        var articles []Article
        err := c.Find(bson.M{}).All(&articles)
        if err != nil {
            ErrorWithJSON(w, "Database error", http.StatusInternalServerError)
            log.Println("Failed get all articles: ", err)
            return
        }

        respBody, err := json.MarshalIndent(articles, "", "  ")
        if err != nil {
            log.Fatal(err)
        }

        ResponseWithJSON(w, respBody, http.StatusOK)
    }
}

func articleByCodigo(s *mgo.Session) func(w http.ResponseWriter, r *http.Request) {  
    return func(w http.ResponseWriter, r *http.Request) {
        session := s.Copy()
        defer session.Close()

        articulo := pat.Param(r, "articulo")

        c := session.DB(database).C(collection)

        var article Article
        err := c.Find(bson.M{"articulo": articulo}).One(&article)
        if err != nil {
            ErrorWithJSON(w, "Database error", http.StatusInternalServerError)
            log.Println("Failed find book: ", err)
            return
        }

        if article.Articulo == "" {
            ErrorWithJSON(w, "Article not found", http.StatusNotFound)
            return
        }

        respBody, err := json.MarshalIndent(article, "", "  ")
        if err != nil {
            log.Fatal(err)
        }

        ResponseWithJSON(w, respBody, http.StatusOK)
    }
}

func articlesByRubro(s *mgo.Session) func(w http.ResponseWriter, r *http.Request) {  
    return func(w http.ResponseWriter, r *http.Request) {
        session := s.Copy()
        defer session.Close()

        rubro := pat.Param(r, "rubro")

        c := session.DB(database).C(collection)

        article := Article{}
        find := c.Find(bson.M{"rubro": rubro})
        items := find.Iter()
        for items.Next(&article) {
            fmt.Println(article.Articulo)
        }

        /*
        if err != nil {
            ErrorWithJSON(w, "Database error", http.StatusInternalServerError)
            log.Println("Failed find book: ", err)
            return
        }

        if article.Articulo == "" {
            ErrorWithJSON(w, "Article not found", http.StatusNotFound)
            return
        }

        respBody, err := json.MarshalIndent(article, "", "  ")
        if err != nil {
            log.Fatal(err)
        }

        ResponseWithJSON(w, respBody, http.StatusOK)
        */
    }
}

func addArticle(s *mgo.Session) func(w http.ResponseWriter, r *http.Request) {  
    return func(w http.ResponseWriter, r *http.Request) {
        session := s.Copy()
        defer session.Close()

        var article Article
        decoder := json.NewDecoder(r.Body)
        err := decoder.Decode(&article)
        if err != nil {
            ErrorWithJSON(w, "Incorrect body", http.StatusBadRequest)
            return
        }

        c := session.DB(database).C(collection)

        err = c.Insert(article)
        if err != nil {
            if mgo.IsDup(err) {
                ErrorWithJSON(w, "Article with this code already exists", http.StatusBadRequest)
                return
            }

            ErrorWithJSON(w, "Database error", http.StatusInternalServerError)
            log.Println("Failed insert article: ", err)
            return
        }

        w.Header().Set("Content-Type", "application/json")
        w.Header().Set("Location", r.URL.Path+"/"+article.Articulo)
        w.WriteHeader(http.StatusCreated)
    }
}

func updateArticle(s *mgo.Session) func(w http.ResponseWriter, r *http.Request) {  
    return func(w http.ResponseWriter, r *http.Request) {
        session := s.Copy()
        defer session.Close()

        articulo := pat.Param(r, "articulo")

        var article Article
        decoder := json.NewDecoder(r.Body)
        err := decoder.Decode(&article)
        if err != nil {
            ErrorWithJSON(w, "Incorrect body", http.StatusBadRequest)
            return
        }

        c := session.DB(database).C(collection)

        err = c.Update(bson.M{"articulo": articulo}, &article)
        if err != nil {
            switch err {
            default:
                ErrorWithJSON(w, "Database error", http.StatusInternalServerError)
                log.Println("Failed update article: ", err)
                return
            case mgo.ErrNotFound:
                ErrorWithJSON(w, "Article not found", http.StatusNotFound)
                return
            }
        }

        w.WriteHeader(http.StatusNoContent)
    }
}

func deleteArticle(s *mgo.Session) func(w http.ResponseWriter, r *http.Request) {  
    return func(w http.ResponseWriter, r *http.Request) {
        session := s.Copy()
        defer session.Close()

        article := pat.Param(r, "article")

        c := session.DB(database).C(collection)

        err := c.Remove(bson.M{"article": article})
        if err != nil {
            switch err {
            default:
                ErrorWithJSON(w, "Database error", http.StatusInternalServerError)
                log.Println("Failed delete article: ", err)
                return
            case mgo.ErrNotFound:
                ErrorWithJSON(w, "Article not found", http.StatusNotFound)
                return
            }
        }

        w.WriteHeader(http.StatusNoContent)
    }
}
