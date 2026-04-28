package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/freehandle/iu/credentials"
)

func NovaMucua(ctx context.Context, app *Aplicacao, port int, caminho string) {
	go app.Rodar(ctx)

	//mux := http.NewServeMux()
	endpoints := map[string]string{
		"signin":      "signin",
		"credentials": "catraca",
		"login":       "credenciais",
	}
	credentialsMux := credentials.New(app.Gerente, endpoints, app.templates, app.NomeMucua, true, "")
	mux := credentialsMux.Mux

	staticPath := fmt.Sprintf("%s/static/", caminho)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticPath))))

	//credentialsMux = credentials.New(app.Gerente, endpoints, app.templates, app.NomeMucua, true, "")
	mux.HandleFunc("/", app.ManejoInicio)
	//mux.HandleFunc("/signin/", app.ManejoSignin)
	mux.HandleFunc("/novousuario", app.ManejoNovoUsuario)
	//mux.HandleFunc("/credenciais", app.ManejoCredenciais)
	//mux.HandleFunc("/catraca", app.ManejoCatraca)
	mux.HandleFunc("/postagem", app.ManejoPostagem)
	mux.HandleFunc("/jornal/", app.ManejoJornal)
	mux.HandleFunc("/publica", app.ManejoPublica)
	mux.HandleFunc("/sair", app.ManejoSair)
	mux.HandleFunc("/conteudo/", app.ManejoConteudo)
	mux.HandleFunc("/optin", app.ManejoOptIn)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      mux,
		WriteTimeout: 5 * time.Second,
	}
	srv.ListenAndServe()
}
