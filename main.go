package main

import (
	"fmt"
	"github.com/kataras/iris/v12"

	prometheusMiddleware "github.com/iris-contrib/middleware/prometheus"
	"github.com/kataras/iris/v12/middleware/logger"
	"github.com/kataras/iris/v12/middleware/recover"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	cnf "github.com/MSJantana/zabbix-exporter-3000/config"
	hdl "github.com/MSJantana/zabbix-exporter-3000/handlers"
)

func main() {
	// Banner color pode ser mantido, opcional
	fmt.Println("\033[31m\r\n\r\n\r\n███████╗███████╗██████╗  ██████╗  ██████╗  ██████╗ \r\n╚══███╔╝██╔════╝╚════██╗██╔═████╗██╔═████╗██╔═████╗ \r\n  ███╔╝ █████╗   █████╔╝██║██╔██║██║██╔██║██║██╔██║ \r\n ███╔╝  ██╔══╝   ╚═══██╗████╔╝██║████╔╝██║████╔╝██║ \r\n███████╗███████╗██████╔╝╚██████╔╝╚██████╔╝╚██████╔╝ \r\n╚══════╝╚══════╝╚═════╝  ╚═════╝  ╚═════╝  ╚═════╝  \r\n\033[m")
	fmt.Println("\033[33mZabbix Exporter for Prometheus")
	fmt.Println("version  : 0.5")
	fmt.Println("Author   : rzrbld")
	fmt.Println("License  : MIT")
	fmt.Println("Git-repo : https://github.com/MSJantana/zabbix-exporter-3000 \033[m \r\n")

	app := iris.New()

	app.Logger().SetLevel("info") // letras minúsculas são recomendadas

	// Middleware para tratar panics e logging
	app.Use(recover.New())
	app.Use(logger.New())

	// Configurando middleware Prometheus
	m := prometheusMiddleware.New("ze3000", 0.3, 1.2, 5.0)
	hdl.RecordMetrics()
	app.Use(m.ServeHTTP)

	// Rota para expor métricas prometheus
	app.Get(cnf.MetricUriPath, iris.FromStd(promhttp.Handler()))

	// Endpoints de saúde
	app.Get("/liveness", func(ctx iris.Context) {
		ctx.WriteString("ok")
	})

	app.Get("/readiness", func(ctx iris.Context) {
		ctx.WriteString("ok")
	})

	// Endpoint raiz simples
	app.Get("/", func(ctx iris.Context) {
		ctx.WriteString("zabbix-exporter-3000")
	})

	// Rodando servidor Iris com porta e sem erro de fechamento
	err := app.Run(iris.Addr(cnf.MainHostPort), iris.WithoutServerError(iris.ErrServerClosed))
	if err != nil {
		app.Logger().Fatalf("Failed to start server: %v", err)
	}
}
