const ApiDocsMixin = {
    data() {
        return {
            sections: [
                {
                    title: i18n("pages.apiDocs.section.auth"),
                    items: [
                        {
                            method: "HEADER",
                            path: "Authorization: Bearer <token>",
                            desc: i18n("pages.apiDocs.tokenHeader"),
                            headers: ["Authorization: Bearer <token>", "или X-API-Token: <token>"],
                            example: `curl -H "Authorization: Bearer <token>" https://<host>/panel/api/inbounds/list`
                        },
                    ]
                },
                {
                    title: "Inbounds",
                    items: [
                        {
                            method: "GET",
                            path: "/panel/api/inbounds/list",
                            desc: "Список инбаундов.",
                            example: `curl -H "Authorization: Bearer <token>" \\
  https://<host>/panel/api/inbounds/list`
                        },
                        {
                            method: "GET",
                            path: "/panel/api/inbounds/get/:id",
                            desc: "Получить инбаунд по ID.",
                            example: `curl -H "Authorization: Bearer <token>" \\
  https://<host>/panel/api/inbounds/get/12`
                        },
                        {
                            method: "POST",
                            path: "/panel/api/inbounds/add",
                            desc: "Добавить инбаунд.",
                            headers: ["Content-Type: application/json"],
                            body: `{
  "remark": "demo",
  "protocol": "vless",
  "port": 443,
  "settings": "{...}",
  "streamSettings": "{...}",
  "sniffing": "{...}"
}`,
                            example: `curl -X POST -H "Authorization: Bearer <token>" \\
  -H "Content-Type: application/json" \\
  -d @inbound.json \\
  https://<host>/panel/api/inbounds/add`
                        },
                        {
                            method: "POST",
                            path: "/panel/api/inbounds/update/:id",
                            desc: "Обновить инбаунд по ID.",
                            headers: ["Content-Type: application/json"],
                            example: `curl -X POST -H "Authorization: Bearer <token>" \\
  -H "Content-Type: application/json" \\
  -d @inbound.json \\
  https://<host>/panel/api/inbounds/update/12`
                        },
                        {
                            method: "POST",
                            path: "/panel/api/inbounds/del/:id",
                            desc: "Удалить инбаунд по ID.",
                            example: `curl -X POST -H "Authorization: Bearer <token>" \\
  https://<host>/panel/api/inbounds/del/12`
                        },
                    ],
                },
                {
                    title: "Server",
                    items: [
                        {
                            method: "GET",
                            path: "/panel/api/server/status",
                            desc: "Статус сервера и XRAY.",
                            example: `curl -H "Authorization: Bearer <token>" \\
  https://<host>/panel/api/server/status`
                        },
                        {
                            method: "GET",
                            path: "/panel/api/server/logs",
                            desc: "Логи панели.",
                            example: `curl -H "Authorization: Bearer <token>" \\
  https://<host>/panel/api/server/logs`
                        },
                        {
                            method: "POST",
                            path: "/panel/api/server/restartXrayService",
                            desc: "Перезапустить XRAY сервис.",
                            example: `curl -X POST -H "Authorization: Bearer <token>" \\
  https://<host>/panel/api/server/restartXrayService`
                        },
                    ],
                },
                {
                    title: "Backup",
                    items: [
                        {
                            method: "GET",
                            path: "/panel/api/backuptotgbot",
                            desc: "Отправить бэкап в Telegram.",
                            example: `curl -H "Authorization: Bearer <token>" \\
  https://<host>/panel/api/backuptotgbot`
                        },
                    ]
                }
            ]
        };
    },
    methods: {
        renderItem() { },
    }
};
