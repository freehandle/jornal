package app

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

func NovaMucua(ctx context.Context, app *Aplicacao, port int, caminho string) {
	go app.Rodar(ctx)

	mux := http.NewServeMux()

	staticPath := fmt.Sprintf("%s/static/", caminho)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticPath))))

	mux.HandleFunc("/signin/", app.ManejoSignin)
	mux.HandleFunc("/novousuario", app.ManejoNovoUsuario)
	mux.HandleFunc("/credenciais", app.ManejoCredenciais)
	mux.HandleFunc("/catraca", app.ManejoCatraca)
	mux.HandleFunc("/postagem", app.ManejoPostagem)
	mux.HandleFunc("/jornal/", app.ManejoJornal)
	mux.HandleFunc("/publica", app.ManejoPublica)
	mux.HandleFunc("/sair", app.ManejoSair)
	mux.HandleFunc("/conteudo/", app.ManejoConteudo)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      mux,
		WriteTimeout: 5 * time.Second,
	}
	srv.ListenAndServe()
}
