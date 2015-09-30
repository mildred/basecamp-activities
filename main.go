package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/mildred/basecamp-to-hipchat/Godeps/_workspace/src/github.com/andybons/hipchat"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
	"strconv"
	"regexp"
)

type APIClient struct {
	http     http.Client
	Username string
	Password string
}

type (
	Account struct {
		Id      int    `json:"id"`
		Name    string `json:"name"`
		Href    string `json:"href"`
		Product string `json:"product"`
	}

	Event struct {
		Id        int          `json:"id"`
		Action    string       `json:"action"`
		Summary   string       `json:"summary"`
		CreatedAt time.Time    `json:"created_at"`
		UpdatedAt time.Time    `json:"updated_at"`
		Bucket    EventBucket  `json:"bucket"`
		HTMLUrl   string       `json:"html_url"`
		Excerpt   string       `json:"excerpt"`
		Creator   EventCreator `json:"creator"`
	}

	EventBucket struct {
		Name   string `json:"name"`
		AppURL string `json:"app_url"`
	}

	EventCreator struct {
		Name string `json:"name"`
	}

	Person struct {
		Id    int    `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email_address"`
		Admin bool   `json:"admin"`
	}

	Project struct {
		Id          int    `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Archived    bool   `json:"archived"`
		Starred     bool   `json:"starred"`
	}

	Todo struct {
		Id       int       `json:"id"`
		Content  string    `json:"content"`
		DueAt    string    `json:"due_at"`
		Comments []Comment `json:"comments"`
		Assignee struct {
			Type string `json:"type"`
			Id   int    `json:"id"`
			Name string `json:"name"`
		} `json:"assignee"`
	}

	TodoList struct {
		Id             int    `json:"id"`
		Name           string `json:"name"`
		Description    string `json:"description"`
		Completed      bool   `json:"completed"`
		CompletedCount int    `json:"completed_count"`
		RemainingCount int    `json:"remaining_count"`
		ProjectId      int    `json:"project_id"`

		Bucket struct {
			Id   int    `json:"id"`
			Name string `json:"name"`
			Type string `json:"type"`
		}

		Todos struct {
			Remaining []*Todo `json:"remaining"`
			Completed []*Todo `json:"completed"`
		}
	}

	Comment struct {
		Id        int       `json:"id"`
		Content   string    `json:"content"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Creator   Person    `json:"creator"`
	}

	Topic struct {
		Id        int       `json:"id"`
		Title     string    `json:"title"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`

		Topicable struct {
			Id   int    `json:"id"`
			Type string `json:"type"`
		} `json:"topicable"`
	}

	Message struct {
		Id        int       `json:"id"`
		Subject   string    `json:"subject"`
		Content   string    `json:"content"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
	}
)

func (api *APIClient) newRequest(account int, method, path string) (*http.Request, error) {
	url := fmt.Sprintf("https://basecamp.com/%d/api/v1%s", account, path)
	//log.Println(url)
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(api.Username, api.Password)
	req.Header.Set("User-Agent", "basecamp-activities (shanti+basecamp@sogilis.com)")
	return req, nil
}

func accountUrl(account int, path string) string {
	return fmt.Sprintf("https://basecamp.com/%d/api/v1%s", account, path)
}

func projectUrl(account, project int, path string) string {
	return fmt.Sprintf("https://basecamp.com/%d/api/v1/projects/%d%s", account, project, path)
}

func (api *APIClient) request(method, url string, result interface{}) error {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(api.Username, api.Password)
	req.Header.Set("User-Agent", "basecamp-activities (shanti+basecamp@sogilis.com)")

	res, err := api.http.Do(req)
	if err != nil {
		return err
	}

	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	//log.Println(url)
	//log.Println(string(bytes))

	return json.Unmarshal(bytes, result)
}

func (api *APIClient) projects(account int) ([]Project, error) {
	var result []Project
	err := api.request("GET", accountUrl(account, "/projects.json"), &result)
	return result, err
}

func (api *APIClient) projectTopics(account, project int) ([]Topic, error) {
	var result []Topic
	err := api.request("GET", projectUrl(account, project, "/topics.json"), &result)
	return result, err
}

func (api *APIClient) projectTodoLists(account, project int) ([]TodoList, error) {
	var result []TodoList
	err := api.request("GET", projectUrl(account, project, "/todolists.json"), &result)
	return result, err
}

func (api *APIClient) projectTodoList(account, project, todolist int) (*TodoList, error) {
	var result *TodoList
	err := api.request("GET", projectUrl(account, project, fmt.Sprintf("/todolists/%d.json", todolist)), &result)
	return result, err
}

func (api *APIClient) projectTodoListRemaining(account, project, todolist int) ([]Todo, error) {
	var result []Todo
	err := api.request("GET", projectUrl(account, project, fmt.Sprintf("/todolists/%d/todos/remaining.json", todolist)), &result)
	return result, err
}

func (api *APIClient) projectTodo(account, project, todo int) (*Todo, error) {
	var result *Todo
	err := api.request("GET", projectUrl(account, project, fmt.Sprintf("/todos/%d.json", todo)), &result)
	return result, err
}

func (api *APIClient) projectMessage(account, project, message int) (*Message, error) {
	var result *Message
	err := api.request("GET", projectUrl(account, project, fmt.Sprintf("/messages/%d.json", message)), &result)
	return result, err
}

func getRoom(basecampProject string, rooms []hipchat.Room) (room *hipchat.Room, isDefault bool) {
	var defaultRoom hipchat.Room
	hasDefault := false
	for _, room := range rooms {
		if basecampProject == room.Name || (room.Topic != "" && strings.Contains(room.Topic, basecampProject)) {
			//log.Printf("Project %s: Choose room %s (topic: %s)", basecampProject, room.Name, room.Topic);
			return &room, false
		} else if strings.Index(room.Topic, "Basecamp:*") >= 0 {
			defaultRoom = room
			hasDefault = true
			//log.Printf("Project %s: Choose default room %s (topic: %s) %d", basecampProject, defaultRoom.Name, defaultRoom.Topic, strings.Index(defaultRoom.Topic, "Basecamp:*"));
		}
	}
	if !hasDefault {
		return nil, false
	}
	return &defaultRoom, true
}

func findProjects(api *APIClient, account, project int, todoMatching *regexp.Regexp, since time.Time) error {
	todolists, err := api.projectTodoLists(account, project)
	if err != nil {
		return err
	}

	for _, todolist := range todolists {
		if todoMatching.MatchString(todolist.Name) {
			remaining, err := api.projectTodoListRemaining(account, project, todolist.Id)
			if err != nil {
				return err
			}
			log.Println(todolist.Name)
			for _, todo := range remaining {
				fullTodo, err := api.projectTodo(account, project, todo.Id)
				if err != nil {
					return err
				}

				var report map[string][]string = map[string][]string{}
				var newReport map[string]bool = map[string]bool{}
				var newMessages int
				update := regexp.MustCompile(`^(?i)Update\s*\:`)
				updateItem := regexp.MustCompile(`^\s*([^\n:]*[^\n:\s])\s*:\s*(.*)\s*$`)
				notAvailable := regexp.MustCompile(`^[Nn]/?[Aa]$`)

				for _, comment := range fullTodo.Comments {
					if ! comment.CreatedAt.Before(since) {
						newMessages += 1
					}
					if update.MatchString(comment.Content) {
						for _, up := range strings.Split(comment.Content, "<br><br>") {
							up = strings.Replace(up, "<br>", "\n", -1)
							ups := updateItem.FindStringSubmatch(up)
							//log.Printf("update: %#v %#v", up, ups)
							if ups != nil {
								key := ups[1]
								val := ups[2]
								if val != "" {
									if notAvailable.MatchString(val) {
										val = ""
									}
									if comment.CreatedAt.Before(since) {
										report[key] = []string{ val }
									} else if newReport[key] {
										if val != "" {
											report[key] = append(report[key], val)
										}
									} else {
										if val == "" {
											delete(report, key)
										} else {
											report[key] = []string{ val }
											newReport[key] = true
										}
									}
								}
							}
						}
					}
				}

				log.Printf("  %s (%s, %d messages, %d new)", todo.Content, fullTodo.Assignee.Name, len(fullTodo.Comments), newMessages)
				//log.Printf("    report = %#v", report)
				for k, vals := range report {
					if len(vals) == 0 {
					} else if len(vals) == 1 {
						log.Printf("    %s: %s", k, vals[0])
					} else {
						log.Printf("    %s:", k)
						for _, v := range vals {
						log.Printf("      - %s", v)
						}
					}
				}

			}
		}
	}

	return nil
}

func lastReport(api *APIClient, account, project int, reportMatching *regexp.Regexp) (*Message, error) {
	topics, err := api.projectTopics(account, project)
	if err != nil {
		return nil, err
	}

	var lastTime time.Time
	var lastMessage int

	for _, topic := range topics {
		if reportMatching.MatchString(topic.Title) {
			if topic.CreatedAt.After(lastTime) && topic.Topicable.Type == "Message" {
				lastTime = topic.CreatedAt
				lastMessage = topic.Topicable.Id
			}
		}
	}

	return api.projectMessage(account, project, lastMessage)
}

func run(basecampAccountId, basecampProjectId int, basecampUser, basecampPass, basecampTodoMatching, basecampReportMatching, hipchatAPIKey string, sleepTime time.Duration) error {

	todoMatching, err := regexp.Compile(basecampTodoMatching)
	if err != nil {
		return err
	}

	reportMatching, err := regexp.Compile(basecampReportMatching)
	if err != nil {
		return err
	}

	api := &APIClient{
		Username: basecampUser,
		Password: basecampPass,
	}

	hipchatClient := hipchat.NewClient(hipchatAPIKey)

	_ = hipchatClient

	last, err := lastReport(api, basecampAccountId, basecampProjectId, reportMatching)
	if err != nil {
		return err
	}

	return findProjects(api, basecampAccountId, basecampProjectId, todoMatching, last.CreatedAt)
	/*
	var c <-chan interface{} = api.monitorEvents(basecampAccountId, sleepTime, time.Now())
	for val := range c {
		if ev, ok := val.(*Event); ok {
			//log.Printf("%v: %v", ev.Bucket.Name, ev.Summary)
			rooms, err := hipchatClient.RoomList()
			if err != nil {
				log.Println(err)
			} else if room, defaultRoom := getRoom(ev.Bucket.Name, rooms); room != nil {
				var message string;
				if defaultRoom {
					message = fmt.Sprintf(
						`<strong><a href="%s">%s</a>, <a href="%s">%s</a></strong><br/>%s`,
						ev.Bucket.AppURL, ev.Bucket.Name, ev.HTMLUrl, ev.Summary, ev.Excerpt)
				} else {
					message = fmt.Sprintf(
						`<strong><a href="%s">%s</a></strong><br/>%s`,
						ev.HTMLUrl, ev.Summary, ev.Excerpt)
				}
				req := hipchat.MessageRequest{
					RoomId:        fmt.Sprintf("%d", room.Id),
					From:          ev.Creator.Name,
					Message:       message,
					Color:         hipchat.ColorPurple,
					MessageFormat: hipchat.FormatHTML,
					Notify:        true,
				}
				if err := hipchatClient.PostMessage(req); err != nil {
					log.Println(err)
				} else {
					//log.Printf("Message sent to room %s", room.Name)
				}
			} else {
				log.Printf("Cannot find a room for %s", ev.Bucket.Name)
			}
		} else {
			log.Println(val)
		}
	}
	*/
	return nil
}

func GetenvInt(varname string, defaultVal int) int {
	value := os.Getenv(varname)
	intVal, err := strconv.ParseInt(value, 10, 0)
	if value == "" || err != nil {
		return defaultVal
	} else {
		return int(intVal)
	}
}

func GetenvStr(varname string, defaultVal string) string {
	value := os.Getenv(varname)
	if value == "" {
		value = defaultVal
	}
	return value
}

func main() {
	var basecampUser = flag.String("basecamp-user", os.Getenv("BASECAMP_USER"), "Username of special basecamp account that can access all projects")
	var basecampPass = flag.String("basecamp-pass", os.Getenv("BASECAMP_PASS"), "Password of special basecamp account that can access all projects")
	var basecampAccountId = flag.Int("basecamp-account", GetenvInt("BASECAMP_ACCOUNT", 0), "Basecamp Account ID")
	var basecampProjectId = flag.Int("basecamp-project", GetenvInt("BASECAMP_PROJECT", 0), "Basecamp project ID")
	var basecampTodoMatching = flag.String("basecamp-todo-matching", GetenvStr("BASECAMP_TODO_MATCHING", "^(Projet|Avant-vente)"), "Regexp that TODO lists must match")
	var basecampReportMatching = flag.String("basecamp-report-matching", GetenvStr("BASECAMP_REPORT_MATCHING", "^Point d'activité"), "Regexp that reports must match")
	var HipchatAPIKey = flag.String("hipchat-api-key", os.Getenv("HIPCHAT_API_KEY"), "API Key for Hipchat")
	var refresh = flag.Duration("refresh", 10*time.Second, "Refresh period for basecamp monitoring")

	flag.Parse()

	err := run(*basecampAccountId, *basecampProjectId, *basecampUser, *basecampPass, *basecampTodoMatching, *basecampReportMatching, *HipchatAPIKey, *refresh)
	if err != nil {
		log.Fatalln(err)
		os.Exit(1)
	}
}
