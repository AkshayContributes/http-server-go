package main

import (
	"bytes"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
)

const (
	HTTP_STATUS_OK    string = "HTTP/1.1 200 OK"
	HTTP_NOT_FOUND    string = "HTTP/1.1 404 NOT FOUND"
	CONTENT_TYPE_TEXT string = "Content-Type: text/plain"
	CONTENT_TYPE      string = "Content-Type: "
	CONTENT_LENGTH    string = "Content-Length: "
	CRLF              string = "\r\n"
)

var dir string

type Request struct {
	Method  string
	Headers map[string]string
	Path    string
	Body    []byte
	Version string
}

type Response struct {
	Headers map[string]string
	Body    []byte
	Status  string
	Version string
}

func writeResponse(res Response, conn net.Conn) {
	_, err := conn.Write([]byte(fmt.Sprintf(res.Version + " " + res.Status + CRLF)))
	if err != nil {
		fmt.Println("Error writing to connection: ", err.Error())
		os.Exit(1)
	}

	for k, v := range res.Headers {
		_, err = conn.Write([]byte(fmt.Sprintf("%s: %s%s", k, v, CRLF)))
		if err != nil {
			fmt.Println("Error writing to connection: ", err.Error())
			os.Exit(1)
		}
	}

	_, err = conn.Write([]byte(CRLF))
	if err != nil {
		fmt.Println("Error writing to connection: ", err.Error())
		os.Exit(1)
	}

	_, err = conn.Write(res.Body)
	if err != nil {
		fmt.Println("Error writing to connection: ", err.Error())
		os.Exit(1)
	}
}

func handleGetRequest(req Request, conn net.Conn) (res Response) {
	res.Headers = make(map[string]string)
	res.Version = req.Version
	path := req.Path

	if strings.HasPrefix(path, "/user-agent") {
		for k, v := range req.Headers {
			if k == "User-Agent" {
				res.Status = "200 OK"
				res.Body = []byte(v)
				res.Headers["Content-Type"] = "text/plain"
				res.Headers["Content-Length"] = fmt.Sprintf("%d", len(res.Body))
				return
			}
		}
	} else if strings.HasPrefix(path, "/files/") {

		fileName := strings.TrimPrefix(path, "/files/")

		file, err := os.ReadFile(dir + fileName)

		if err != nil {
			res.Status = "404 Not Found"
			res.Body = []byte("")
		} else {
			res.Body = file
			res.Status = "200 OK"
			res.Headers["Content-Type"] = "application/octet-stream"
			res.Headers["Content-Length"] = fmt.Sprintf("%d", len(res.Body))
			res.Headers["Connection"] = "close"
		}
		return
	} else if strings.HasPrefix(path, "/echo/") {
		res.Status = "200 OK"
		res.Body = []byte(strings.TrimPrefix(path, "/echo/"))
		res.Headers["Content-Type"] = "text/plain"
		res.Headers["Content-Length"] = fmt.Sprintf("%d", len(res.Body))
		return
	} else if path == "/" {
		res.Status = "200 OK"
		res.Body = []byte("")
		res.Headers["Content-Type"] = "text/plain"
		res.Headers["Content-Length"] = fmt.Sprintf("%d", len(res.Body))
		return
	} else {
		res.Status = "404 Not Found"
		res.Body = []byte("")
	}
	return
}

func handlePostRequest(req Request, conn net.Conn) (res Response) {
	res.Headers = make(map[string]string)
	res.Version = req.Version
	path := req.Path

	if strings.HasPrefix(path, "/files/") {
		fname := strings.TrimPrefix(path, "/files/")
		fp := fmt.Sprint(dir, string(os.PathSeparator), fname)
		err := os.WriteFile(fp, req.Body, 0666)
		if err != nil {
			res.Status = "404 Not Found"
			res.Body = []byte("")
			return
		}
	}

	res.Status = "201 Created"
	res.Body = req.Body
	res.Headers["Content-Type"] = "application/octet-stream"
	res.Headers["Content-Length"] = fmt.Sprintf("%d", len(res.Body))
	res.Headers["Connection"] = "close"

	return

}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	req := parseMyRequest(conn)

	var res Response

	switch req.Method {
	case "GET":
		res = handleGetRequest(req, conn)
	case "POST":
		res = handlePostRequest(req, conn)
	default:
		res = Response{Version: "HTTP/1.1", Status: "404 Not Found", Headers: make(map[string]string), Body: []byte("")}
	}

	writeResponse(res, conn)

}

func parseMyRequest(conn net.Conn) (request Request) {
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		fmt.Println("Error reading from connection: ", err.Error())
		os.Exit(1)
	}

	//turn the slice of bytes into string
	httpRequest := string(buffer[:n])

	//split the request into headers and body
	data := bytes.Split([]byte(httpRequest), []byte("\r\n\r\n"))

	headerStr := string(data[0])
	lines := strings.Split(headerStr, "\r\n")
	firstLine := strings.Fields(lines[0])

	request.Method = firstLine[0]
	request.Path = firstLine[1]
	request.Version = firstLine[2]
	request.Body = data[1]
	request.Headers = make(map[string]string)

	for i, line := range lines {
		if i == 0 {
			continue
		}

		headArr := strings.Split(line, ": ")
		request.Headers[headArr[0]] = headArr[1]
	}

	return request

}

func main() {

	directoryPtr := flag.String("directory", "xxxx", "Directory of the file storage")
	flag.Parse()

	dir = *directoryPtr

	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}
		go handleConnection(conn)
	}

}
