package dim

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

type Client struct {
	endpoint   string
	httpClient *http.Client
	auth       AuthStruct
	token      string
	logger     log.Logger
}

// AuthStruct -
type AuthStruct struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type rawResponse struct {
	Result interface{}      `json:"result"`
	Error  rawResponseError `json:"error"`
}

type rawResponseError struct {
	Code    int64  `json:"code"`
	Message string `json:"message"`
}

//type Response interface{}

func NewClient(endpoint, token, username, password *string, logger log.Logger) (*Client, error) {

	c := Client{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		endpoint:   *endpoint,
		token:      *token,
		auth: AuthStruct{
			Username: *username,
			Password: *password,
		},
		logger: logger,
	}

	// do login if we have no token
	if c.token == "" {
		err := c.doLogin()
		if err != nil {
			return nil, fmt.Errorf("could not login to DIM, %s", err)
		}
	}

	return &c, nil
}

func (c *Client) doRequest(req *http.Request) ([]byte, error) {

	req.AddCookie(&http.Cookie{Name: "session", Value: c.token})

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status: %d, body: %s", res.StatusCode, body)
	}

	return body, err
}

// obtains token (session cookie) from DIM
func (c *Client) doLogin() error {
	if c.auth.Username == "" || c.auth.Password == "" {
		return fmt.Errorf("define username and password")
	}

	form := url.Values{}
	form.Add("username", c.auth.Username)
	form.Add("password", c.auth.Password)

	res, err := c.httpClient.PostForm(fmt.Sprintf("%s/login", c.endpoint), form)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	c.token = ""
	cookies := res.Cookies()
	for _, cookie := range cookies {
		if cookie.Name == "session" {
			c.token = cookie.Value
			break
		}
	}
	if c.token == "" {
		return fmt.Errorf("could not get session cookie")
	}

	return nil
}

func (c *Client) RawCall(function string, args interface{}) (any, error) {

	/*
		if rArgs := reflect.ValueOf(args); rArgs.Kind() == reflect.Struct {
			args = structToKVList(rArgs)
		}
	*/
	payload := map[string]interface{}{"jsonrpc": "2.0", "method": function, "params": args, "id": nil}
	body, err := json.Marshal(payload)
	if c.logger != nil {
		level.Info(c.logger).Log("msg", "exec dim call", "payload", string(body))
	}
	//fmt.Println(string(body)) //debug
	buf := bytes.NewBuffer(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/jsonrpc", c.endpoint), buf)
	if err != nil {
		return nil, err
	}

	resBody, err := c.doRequest(req)
	if err != nil {
		return nil, fmt.Errorf("could not perform DIM request, %s", err)
	}

	var rawRe rawResponse
	// DIM seemingly always returnd Content-Type="text/html", so we have to expect json and unmarshal it.
	// In case of incorrectly specified dim url, it might be a response (without redirect) from SSO page,
	// which will fail unmarshal
	err = json.Unmarshal(resBody, &rawRe)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal DIM response (is specified dim url correct?): %s", err)
	}

	if rawRe.Error.Code != 0 {
		return nil, Error{Func: function, Code: rawRe.Error.Code, Message: rawRe.Error.Message}
	}
	return rawRe.Result, nil
}

// structToKVList coverts arbitrary struct to []map[string]interface{}
// where each element of the slice is a map with single key, value, e.g map[string]interface{"key": 1}
// Primarily used for convirting struct args to dim API args (json array of {"key": value} )
func structToKVList(v reflect.Value) []map[string]interface{} {
	var res []map[string]interface{}
	if v.Kind() == reflect.Struct {
		//		res = make([]map[string]interface{}, v.NumField())
		for i := 0; i < v.NumField(); i++ {
			fieldInfo := v.Type().Field(i)
			tag := fieldInfo.Tag
			jtag := tag.Get("json")
			var name string
			var omitEmpty bool
			if jtag == "" {
				name = fieldInfo.Name
			} else {
				mdata := strings.Split(jtag, ",")
				name = mdata[0]
				if (len(mdata) > 1) && (mdata[1] == "omitempty") {
					omitEmpty = true
				}
			}
			if !(omitEmpty && isEmptyValue(v.Field(i))) {
				res = append(res, map[string]interface{}{name: v.Field(i).Interface()})
			}
		}
	}
	return res
}

// copied from json/encode
func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}
	return false
}
