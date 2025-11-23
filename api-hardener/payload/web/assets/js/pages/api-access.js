const ApiAccessMixin = {
    data() {
        return {
            apiSettings: {
                apiTokenOnly: true,
                apiDefaultRateLimit: 120,
            },
            apiUsers: [],
            apiUserForm: {
                name: "",
                rate: 0,
            },
            apiStates: {
                loading: false,
                saving: false,
                creating: false,
            },
            apiTable: {
                columns: [
                    { title: "#", dataIndex: "id", key: "id", width: 60 },
                    { title: i18n("pages.settings.api.user"), dataIndex: "name", key: "name" },
                    { title: i18n("status"), dataIndex: "status", key: "status", scopedSlots: { customRender: "status" } },
                    { title: i18n("pages.settings.api.rate"), dataIndex: "rateLimitPerMinute", key: "rate", scopedSlots: { customRender: "rate" }, width: 180 },
                    { title: i18n("pages.settings.api.lastUsed"), dataIndex: "lastUsedAt", key: "lastUsedAt", scopedSlots: { customRender: "lastUsed" }, width: 200 },
                    { title: i18n("action"), key: "actions", scopedSlots: { customRender: "actions" }, width: 260 },
                ],
            },
            tokenModal: {
                visible: false,
                token: "",
            },
        };
    },
    methods: {
        async initApiAccess() {
            await Promise.all([this.fetchApiSettings(), this.fetchApiUsers()]);
        },
        async fetchApiSettings() {
            this.apiStates.loading = true;
            const msg = await HttpUtil.get("/panel/api-users/settings");
            this.apiStates.loading = false;
            if (msg && msg.success) {
                this.apiSettings = msg.obj;
            }
        },
        async saveApiSettings() {
            this.apiStates.saving = true;
            const msg = await HttpUtil.post("/panel/api-users/settings", this.apiSettings);
            this.apiStates.saving = false;
            if (msg && msg.success) {
                Vue.prototype.$message.success(i18n("pages.settings.api.settingsUpdated"));
                await this.fetchApiSettings();
            }
        },
        async fetchApiUsers() {
            this.apiStates.loading = true;
            const msg = await HttpUtil.get("/panel/api-users/list");
            this.apiStates.loading = false;
            if (msg && msg.success) {
                this.apiUsers = (msg.obj || []).map(u => ({
                    ...u,
                    key: u.id,
                }));
            }
        },
        async createApiUser() {
            if (!this.apiUserForm.name) {
                Vue.prototype.$message.error(i18n("pages.settings.api.userNameRequired"));
                return;
            }
            this.apiStates.creating = true;
            const msg = await HttpUtil.post("/panel/api-users/create", this.apiUserForm);
            this.apiStates.creating = false;
            if (msg && msg.success) {
                await this.fetchApiUsers();
                this.apiUserForm = { name: "", rate: 0 };
                if (msg.obj && msg.obj.token) {
                    this.tokenModal.token = msg.obj.token;
                    this.tokenModal.visible = true;
                }
            }
        },
        async toggleApiUser(user, enabled) {
            const endpoint = enabled ? "enable" : "disable";
            const msg = await HttpUtil.post(`/panel/api-users/${endpoint}/${user.id}`);
            if (msg && msg.success) {
                Vue.prototype.$message.success(i18n(enabled ? "pages.settings.api.userEnabled" : "pages.settings.api.userDisabled"));
                await this.fetchApiUsers();
            }
        },
        async rotateApiUser(user) {
            const msg = await HttpUtil.post(`/panel/api-users/rotate/${user.id}`);
            if (msg && msg.success && msg.obj && msg.obj.token) {
                this.tokenModal.token = msg.obj.token;
                this.tokenModal.visible = true;
            }
        },
        async deleteApiUser(user) {
            await new Promise(resolve => {
                this.$confirm({
                    title: i18n("pages.settings.api.deleteConfirmTitle"),
                    content: i18n("pages.settings.api.deleteConfirmDesc"),
                    okText: i18n("sure"),
                    cancelText: i18n("cancel"),
                    onOk: () => resolve(true),
                    onCancel: () => resolve(false),
                });
            }).then(async (confirm) => {
                if (!confirm) return;
                const msg = await HttpUtil.post(`/panel/api-users/delete/${user.id}`);
                if (msg && msg.success) {
                    Vue.prototype.$message.success(i18n("pages.settings.api.userDeleted"));
                    await this.fetchApiUsers();
                }
            });
        },
        async updateApiRate(user) {
            const msg = await HttpUtil.post(`/panel/api-users/rate/${user.id}`, { rate: user.rateLimitPerMinute });
            if (msg && msg.success) {
                Vue.prototype.$message.success(i18n("pages.settings.api.rateUpdated"));
                await this.fetchApiUsers();
            }
        },
        copyToken() {
            if (!this.tokenModal.token) return;
            ClipboardManager.copyText(this.tokenModal.token).then(() => {
                Vue.prototype.$message.success(i18n("copySuccess"));
            });
        },
    },
    mounted() {
        this.initApiAccess();
    }
};
