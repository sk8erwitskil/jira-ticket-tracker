package main
/*
  Author: Kyle Laplante

  This is a program that continuously searches for tickets created by a certain
  user in a certain project in Jira and acts upon finding them. Currently, this
  program will only print out the key and summary of the issues it finds. You
  should implement your own function in the `readIssues` function to do
  whatever you want to do with the tickets you find.

  Example:
    ./jira-ticket-tracker --config=./config.yaml --project=MyTeam --user=klapante

  The yaml config should be in the following format:

    login: myuser
    password: mypassword
    url: https://jira.whatever.com/rest/api/2

*/

import (
  "encoding/json"
  "flag"
  "fmt"
  "github.com/plouc/go-jira-client"
  "io/ioutil"
  "launchpad.net/goyaml"
  "log"
  "net/http"
  "os"
  "time"
)

var (
  // command line flags
  config  = flag.String("config", "./config.yaml", "The path to the jira config to connect to")
  project = flag.String("project", "", "The jira project to search for tickets in")
  user    = flag.String("user", "", "The user to search for tickets for")
  // create the logger
  logger  = log.New(os.Stderr, "", log.LstdFlags)
)

const (
  dateLayout       = "2006-01-02T15:04:05.000-0700"
  maxSearchResults = 20          // max number of issues allowed in one search
  trackingMethod   = "reporter"  // either "reporter" or "assignee"
  waitIntervalSecs = 4           // how long to wait between searches
)

// store the credentials in a file outside the code
type Config struct {
  Login    string `yaml:"login"`
  Password string `yaml:"password"`
  Url      string `yaml:"url"`  // e.g. https://jira.whatever.com/rest/api/2
}

func getCreds(configPath string) Config {
  // read the yaml file
  file, err := ioutil.ReadFile(configPath)
  if err != nil {
    logger.Print("Error reading config file: ", err)
    os.Exit(1)  // exit if we cannot read the creds
  }

  // parse the config
  var config Config
  err = goyaml.Unmarshal(file, &config)
  if err != nil {
    logger.Print("Error parsing yaml: ", err)
    os.Exit(1) // exit if we cannot read the creds
  }

  return config
}

func jiraQuery(uri string, creds *Config) (contents []byte) {
  url := creds.Url + uri

  req, err := http.NewRequest("GET", url, nil)
  if err != nil {
    logger.Print("Error making a request to jira: ", err)
    return
  }
  req.SetBasicAuth(creds.Login, creds.Password)

  client := &http.Client{}
  resp, err := client.Do(req)
  defer resp.Body.Close()
  if err != nil {
    logger.Print("Error calling ", url, ": ", err)
    return
  }

  contents, err = ioutil.ReadAll(resp.Body)
  if err != nil {
    logger.Print("Unable to read body contents: ", err)
    return
  }

  return
}

func jiraSearch(field, value string, maxResults int, creds *Config) []byte {
  uri := fmt.Sprintf(
      "/search?jql=%s=%s+order+by+created&startAt=0&maxResults=%d",
      field,
      value,
      maxResults,
  )

  return jiraQuery(uri, creds)
}

func issueFilter(project string, age int) func(i *gojira.Issue) bool {
  return func(i *gojira.Issue) bool {
    t, err := time.Parse(dateLayout, i.Fields.Created)
    if err != nil {
      logger.Print("Error parsing time ", i.Fields.Created, ": ", err)
      return false  // skip this issue if we cannot parse the time
    }
    since := time.Now().UTC().Unix() - t.Unix()
    if since < int64(age) && i.Fields.Project.Key == project {
      return true
    } else {
      return false
    }
  }
}

func recentIssuesFromUser(user, project string, creds *Config) []*gojira.Issue {
  filteredIssues := []*gojira.Issue{}
  issueIsMatch := issueFilter(project, waitIntervalSecs)

  // get the contents of the search
  contents := jiraSearch(trackingMethod, user, maxSearchResults, creds)
  // change "reporter" to "assignee" if you want to track tickets
  // that were assigned TO the user

  // parse the contents into a list of issues
  var issues gojira.IssueList
  err := json.Unmarshal(contents, &issues)
  if err != nil {
    logger.Print("Error parsing json: ", err)
    return filteredIssues
  }

  // scan the issues for ones that match our filter of user/project/age
  for _, issue := range issues.Issues {
    if issueIsMatch(issue) {
      filteredIssues = append(filteredIssues, issue)
    }
  }

  return filteredIssues
}

func waitForIssues(user, project string, creds *Config, c chan *gojira.Issue) {
  for {
    time.Sleep(time.Duration(waitIntervalSecs * time.Second))
    issues := recentIssuesFromUser(user, project, creds)
    for _, issue := range issues {
      c <- issue
    }
  }
}

func readIssues(c chan *gojira.Issue) {
  for {
    issue := <-c
    logger.Print(fmt.Sprintf("Found: [%s] %s", issue.Key, issue.Fields.Summary))
    /*
       implement your own functions here
       to do whatever you want with the issues
       that are found. in the current state this
       program will only print the ticket key and
       summary when one is found.
    */
  }
}

func main() {
  flag.Parse()

  if len(*project) == 0 {
    // project is required
    logger.Print("Please specify a project")
    os.Exit(1)
  } else if len(*user) == 0 {
    // user is required
    logger.Print("Please specify a user")
    os.Exit(1)
  }
  logger.Print("Searching in [", *project, "] for ", *user)

  creds := getCreds(*config)

  c := make(chan *gojira.Issue)
  // create the producer
  go waitForIssues(*user, *project, &creds, c)
  // create the consumer
  go readIssues(c)

  // so the program wont end
  var input string
  fmt.Scanln(&input)
}
