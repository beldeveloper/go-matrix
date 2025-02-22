# go-matrix

## Description

A Golang library that provides the API-based SDK to interact with the Matrix server.

## Installation

```shell
go get github.com/beldeveloper/go-matrix
```

## Example

```go
// auth
client, err := gomatrix.NewClient(gomatrix.Credentials{
    Server:   "https://matrix.o",
    User:     "<matrix_user>",
    Password: "<matrix_password>",
})

// send text message
err := client.SendText(ctx, "<room_id>", "<text_message>")

// send media
mediaURI, err := matrix.UploadFile(ctx, "image/jpeg", fileData)

err = matrix.SendMedia(ctx, "<room_id>", gomatrix.Media{
    Type:    gomatrix.Image,
    Caption: "<caption>",
    URI:     mediaURI,
})
```