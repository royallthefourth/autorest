package bartlett

import (
	"encoding/json"
	"fmt"
	"github.com/buger/jsonparser"
	"io/ioutil"
	"log"
	"net/http"
)

// Routes generates all of the URLs and handlers for the tables specified in Bartlett.
// Iterate this output to feed it into your web server, prefix or otherwise alter the route names,
// and add filtering to the handler functions.
func (b *Bartlett) Routes() (paths []string, handlers []func(http.ResponseWriter, *http.Request)) {
	paths = make([]string, len(b.Tables))
	handlers = make([]func(http.ResponseWriter, *http.Request), len(b.Tables))
	for i, t := range b.Tables {
		columns, err := b.Driver.GetColumns(b.DB, t)
		if err != nil {
			log.Println(err.Error())
		} else {
			t.columns = columns
		}
		paths[i] = fmt.Sprintf(`/%s`, t.Name)
		handlers[i] = b.handleRoute(t)
	}

	return paths, handlers
}

func (b Bartlett) handleRoute(t Table) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(`Content-Type`, `application/json`)

		switch r.Method {
		case http.MethodGet:
			b.handleGet(t, w, r)
		case http.MethodPost:
			b.handlePost(t, w, r)
		case http.MethodDelete:
			b.handleDelete(t, w, r)
		case http.MethodPatch:
			b.handlePatch(t, w, r)
		default:
			w.WriteHeader(http.StatusNotImplemented)
			return
		}
	}
}

func (b Bartlett) handleDelete(t Table, w http.ResponseWriter, r *http.Request) {
	// TODO verify b.validateWrite will work here

	if !t.Writable {
		w.WriteHeader(http.StatusMethodNotAllowed)
		log.Println(r.RequestURI + ` Invalid insert attempt to read-only ` + t.Name)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"error": "Table %s is read-only"}`, t.Name)))
		return
	}

	query, err := b.buildUpdate(t, r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Println(r.RequestURI + err.Error())
		return
	}

	rows, err := query.RunWith(b.DB).Query()
	defer rows.Close()

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(r.RequestURI + err.Error())
		return
	}

	err = b.Driver.MarshalResults(rows, w)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(r.RequestURI + err.Error())
		return
	}
}

func (b Bartlett) handleGet(t Table, w http.ResponseWriter, r *http.Request) {
	query, err := b.buildSelect(t, r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(r.RequestURI + err.Error())
		return
	}

	rows, err := query.RunWith(b.DB).Query()
	defer rows.Close()

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(r.RequestURI + err.Error())
		return
	}

	err = b.Driver.MarshalResults(rows, w)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(r.RequestURI + err.Error())
		return
	}
}

func (b Bartlett) handlePatch(t Table, w http.ResponseWriter, r *http.Request) {
	// TODO use b.validateWrite

	if !t.Writable {
		w.WriteHeader(http.StatusMethodNotAllowed)
		log.Println(r.RequestURI + ` Invalid insert attempt to read-only ` + t.Name)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"error": "Table %s is read-only"}`, t.Name)))
		return
	}

	query, err := b.buildUpdate(t, r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(r.RequestURI + err.Error())
		return
	}

	rows, err := query.RunWith(b.DB).Query()
	defer rows.Close()

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(r.RequestURI + err.Error())
		return
	}

	err = b.Driver.MarshalResults(rows, w)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(r.RequestURI + err.Error())
		return
	}
}

func (b Bartlett) handlePost(t Table, w http.ResponseWriter, r *http.Request) {
	status, userID, err := b.validateWrite(t, r)
	if err != nil {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"error": "%s"}`, err.Error())))
		return
	}

	tx, err := b.DB.Begin()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(r.RequestURI + err.Error())
		return
	}

	rawBody, _ := ioutil.ReadAll(r.Body)
	n, err := jsonparser.ArrayEach(rawBody, func(row []byte, dataType jsonparser.ValueType, offset int, err error) {
		query := t.prepareInsert(row, userID)
		_, err = query.RunWith(tx).Exec()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Println(r.RequestURI + err.Error())
			return
		}

	})

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(r.RequestURI + err.Error())
		return
	}

	err = tx.Commit()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(r.RequestURI + err.Error())
		return
	}

	_, _ = w.Write([]byte(fmt.Sprintf(`{"inserts": %d}`, n)))
}

func (b Bartlett) validateWrite(t Table, r *http.Request) (status int, userID interface{}, err error) {
	status = http.StatusOK

	if !t.Writable {
		status = http.StatusMethodNotAllowed
		err = fmt.Errorf(`table %s is read-only`, t.Name)
		return status, nil, err
	}

	if len(t.UserID) > 0 {
		userID, err = b.Users(r)
		if err != nil || userID == nil {
			status = http.StatusForbidden
			err = fmt.Errorf(`failed to generate userID: %s`, err.Error())
			return status, nil, err
		}
	} else {
		userID = 0
	}

	rawBody, err := ioutil.ReadAll(r.Body)
	if rune(rawBody[0]) != '[' || !json.Valid(rawBody) {
		status = http.StatusBadRequest
		err = fmt.Errorf(`JSON data should be an array`)
		return status, userID, err
	}

	return status, userID, err
}
