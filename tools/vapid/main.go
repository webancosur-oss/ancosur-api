package main

import (
	"fmt"
	"log"

	webpush "github.com/SherClockHolmes/webpush-go"
)

/*
Genera las claves VAPID para Web Push.
Uso: go run ./tools/vapid
Copiar el resultado a las variables de Railway.
Nunca guardarlas en el repo.
*/
func main() {
	privateKey, publicKey, err := webpush.GenerateVAPIDKeys()

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("VAPID_PUBLIC_KEY=" + publicKey)
	fmt.Println("VAPID_PRIVATE_KEY=" + privateKey)
	fmt.Println("VAPID_SUBJECT=ventas@ancosur.pe")
}
