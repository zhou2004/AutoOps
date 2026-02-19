<template>
    <div class="login-container">
        <div class="login-card">
            <div class="card-header">
                <h2 class="card-title">AutoOps</h2>
                <span class="card-subtitle">运维管理系统</span>
            </div>

            <!-- 表单 -->
            <el-form ref="loginFormRef" :rules="rules" :model="loginForm">
                <el-form-item prop="username">
                    <el-input
                        v-model="loginForm.username"
                        placeholder="请输入账号"
                        clearable
                        class="dark-input"
                    />
                </el-form-item>

                <el-form-item prop="password">
                    <el-input
                        v-model="loginForm.password"
                        placeholder="请输入密码"
                        type="password"
                        show-password
                        clearable
                        class="dark-input"
                    />
                </el-form-item>

                <el-form-item prop="image">
                    <div class="captcha-row">
                        <el-input
                            v-model="loginForm.image"
                            placeholder="请输入验证码"
                            maxlength="6"
                            clearable
                            class="dark-input"
                        />
                        <div class="captcha-box" @click="getCaptcha">
                            <el-image :src="image" class="captcha-img" />
                        </div>
                    </div>
                </el-form-item>

                <el-form-item>
                    <el-button class="login-btn" type="primary" @click="loginBtn">登 录</el-button>
                    <el-button class="reset-btn" @click="resetLoginForm">重 置</el-button>
                </el-form-item>
            </el-form>
        </div>
    </div>
</template>

<script>
export default {
    name: "Login",
    data() {
        return {
            image: '',
            rules: {
                username: [{ required: true, message: "请输入账号", trigger: "blur" }],
                password: [{ required: true, message: "请输入密码", trigger: "blur" }],
                image:    [{ required: true, message: "请输入验证码", trigger: "blur" }]
            },
            loginForm: {
                username: '',
                password: '',
                image: '',
                idKey: ''
            }
        }
    },
    methods: {
        async getCaptcha() {
            const { data: res } = await this.$api.captcha()
            if (res.code !== 200) {
                this.$message.error(res.message)
            } else {
                this.image = res.data.image
                this.loginForm.idKey = res.data.idKey
            }
        },
        loginBtn() {
            this.$refs.loginFormRef.validate(async valid => {
                if (valid) {
                    const { data: res } = await this.$api.login(this.loginForm)
                    if (res.code !== 200) {
                        this.$message.error(res.message)
                    } else {
                        this.$message.success("登录成功")
                        this.$store.commit('saveSysAdmin', res.data.sysAdmin)
                        this.$store.commit('saveToken', res.data.token)
                        this.$store.commit('saveLeftMenuList', res.data.leftMenuList)
                        this.$store.commit('savePermissionList', res.data.permissionList)
                        await this.$router.push("/home")
                    }
                } else {
                    return false
                }
            })
        },
        resetLoginForm() {
            this.$refs.loginFormRef.resetFields()
        }
    },
    created() {
        this.getCaptcha()
    }
}
</script>

<style lang="less" scoped>
.login-container {
    width: 100%;
    height: 100vh;
    display: flex;
    align-items: center;
    justify-content: center;
    background: url('../assets/image/背景.jpg') center / cover no-repeat;
    font-family: 'PingFang SC', 'Microsoft YaHei', sans-serif;

}

// 卡片
.login-card {
    position: relative;
    z-index: 1;
    width: 400px;
    padding: 40px 36px 32px;
    background: rgba(10, 14, 30, 0.82);
    border: 1px solid rgba(255, 255, 255, 0.12);
    border-radius: 16px;
    box-shadow: 0 24px 64px rgba(0, 0, 0, 0.5);
    text-align: center;
}

// 标题行
.card-header {
    display: flex;
    align-items: baseline;
    justify-content: center;
    gap: 10px;
    margin-bottom: 28px;
}

.card-title {
    font-size: 22px;
    font-weight: 700;
    color: #f1f5f9;
    margin: 0;
    letter-spacing: 1px;
}

.card-subtitle {
    font-size: 22px;
    font-weight: 700;
    color: #f1f5f9;
    letter-spacing: 1px;
}

// 输入框深色风格
.dark-input {
    /deep/ .el-input__inner {
        background: #ffffff !important;
        border: 1px solid rgba(255, 255, 255, 0.2) !important;
        border-radius: 0 !important;
        color: #1a1a1a !important;
        font-size: 14px !important;
        height: 44px !important;
        line-height: 44px !important;
        padding: 0 10px !important;
        transition: border-color 0.25s, box-shadow 0.25s !important;

        &::placeholder {
            color: #aaaaaa !important;
        }

        &:focus {
            border-color: #6366f1 !important;
            box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.15) !important;
        }
    }

    /deep/ .el-input {
        background: transparent !important;
    }

    /deep/ .el-input__prefix { color: rgba(148, 163, 184, 0.6); }
    /deep/ .el-input__clear,
    /deep/ .el-icon-view  { color: rgba(148, 163, 184, 0.5); &:hover { color: #6366f1; } }
}

// 验证码行
.captcha-row {
    display: flex;
    gap: 10px;

    .dark-input { flex: 1; }
}

.captcha-box {
    flex-shrink: 0;
    width: 110px;
    height: 44px;
    border-radius: 8px;
    overflow: hidden;
    cursor: pointer;
    border: 1px solid rgba(255, 255, 255, 0.12);
    transition: border-color 0.25s;

    &:hover { border-color: #6366f1; }
}

.captcha-img {
    width: 100%;
    height: 100%;
    display: block;
}

// 表单间距调整
/deep/ .el-form-item {
    margin-bottom: 18px;

    &:last-child { margin-bottom: 0; }

    .el-form-item__error {
        font-size: 11px;
        color: #f87171;
        padding-top: 3px;
    }
}

// 登录按钮
.login-btn {
    width: calc(100% - 100px);
    height: 44px;
    border: none;
    border-radius: 8px;
    background: linear-gradient(135deg, #6366f1, #8b5cf6) !important;
    color: #fff !important;
    font-size: 15px !important;
    font-weight: 600 !important;
    letter-spacing: 2px;
    box-shadow: 0 4px 16px rgba(99, 102, 241, 0.35) !important;
    transition: all 0.25s !important;

    &:hover {
        transform: translateY(-1px);
        box-shadow: 0 8px 24px rgba(99, 102, 241, 0.5) !important;
    }
    &:active { transform: translateY(0); }
}

// 重置按钮
.reset-btn {
    width: 86px;
    height: 44px;
    margin-left: 10px !important;
    border-radius: 8px;
    border: 1px solid rgba(255, 255, 255, 0.12) !important;
    background: transparent !important;
    color: rgba(148, 163, 184, 0.8) !important;
    font-size: 14px !important;
    letter-spacing: 1px;
    transition: all 0.25s !important;

    &:hover {
        border-color: rgba(255, 255, 255, 0.25) !important;
        color: #e2e8f0 !important;
        background: rgba(255, 255, 255, 0.06) !important;
    }
}
</style>
