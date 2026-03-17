APP_NAME=app.exe

run:
	air -- bento run

build:
	go build -o build/$(APP_NAME) .

clean:
	rm -f build/$(APP_NAME)
