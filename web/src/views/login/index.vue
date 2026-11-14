<script setup lang="ts">
import { computed, onMounted, onBeforeUnmount, reactive, ref } from "vue";
import { useRouter } from "vue-router";
import type { FormInstance, FormRules } from "element-plus";
import {
  Moon,
  Sunny,
  ArrowRight,
  Refresh,
  ArrowLeft,
  Check
} from "@element-plus/icons-vue";
import { tr } from "@/locales";
import LanguageSwitch from "@/components/LanguageSwitch/index.vue";
import BrandLogo from "@/components/BrandLogo/index.vue";
import { message } from "@/utils/message";
import { useLayout } from "@/layout/hooks/useLayout";
import { useDataThemeChange } from "@/layout/hooks/useDataThemeChange";
import { useUserStoreHook } from "@/store/modules/user";
import { usePermissionStoreHook } from "@/store/modules/permission";
import {
  getCaptcha,
  getAccountOptions,
  sendAccountCode,
  registerAccount,
  resetAccountPassword
} from "@/api/user";
import "./auth.css";
defineOptions({ name: "Login" });
const router = useRouter();
useLayout().initStorage();
const { dataTheme, overallStyle, dataThemeChange } = useDataThemeChange();
dataThemeChange(overallStyle.value || "light");
const mode = ref<"login" | "register" | "reset">("login");
const loading = ref(false),
  captchaLoading = ref(false),
  captchaFailed = ref(false);
const captchaImage = ref(""),
  captchaId = ref(""),
  verificationId = ref(""),
  sentEmail = ref(""),
  success = ref("");
const options = reactive({
  registrationEnabled: false,
  emailEnabled: false,
  loaded: false
});
const formRef = ref<FormInstance>();
const form = reactive({
  username: "",
  email: "",
  password: "",
  confirm: "",
  image: "",
  code: ""
});
const stepTwo = computed(
  () => mode.value !== "login" && !!verificationId.value
);
const title = computed(() =>
  tr(
    mode.value === "login"
      ? "欢迎回来"
      : mode.value === "register"
        ? "创建你的账号"
        : "找回账号密码"
  )
);
const caption = computed(() =>
  tr(
    mode.value === "login"
      ? "登录 AutoOps，进入你的运维工作空间。"
      : stepTwo.value
        ? "输入邮件验证码，完成账号验证。"
        : "通过邮箱验证，安全地继续下一步。"
  )
);
const rules = computed<FormRules>(() => ({
  username: [
    { required: true, message: tr("请输入账号"), trigger: "blur" },
    ...(mode.value === "register"
      ? [
          {
            pattern: /^[a-zA-Z][a-zA-Z0-9_-]{2,31}$/,
            message: tr(
              "账号需为 3–32 位字母、数字、下划线或短横线，并以字母开头"
            ),
            trigger: "blur"
          }
        ]
      : [])
  ],
  email: [
    { required: true, message: tr("请输入邮箱"), trigger: "blur" },
    { type: "email", message: tr("请输入有效邮箱"), trigger: "blur" }
  ],
  password: [
    { required: true, message: tr("请输入密码"), trigger: "blur" },
    ...(mode.value === "login"
      ? []
      : [
          {
            validator: (_r, value, done) => {
              const n = new TextEncoder().encode(value).length;
              done(
                n >= 10 && n <= 72
                  ? undefined
                  : new Error(tr("密码长度需为 10–72 字节"))
              );
            },
            trigger: "blur"
          }
        ])
  ],
  confirm: [
    {
      validator: (_r, value, done) =>
        done(
          value && value === form.password
            ? undefined
            : new Error(tr("两次输入的密码不一致"))
        ),
      trigger: "blur"
    }
  ],
  image: [
    {
      required: true,
      pattern: /^\d{4}$/,
      message: tr("请输入 4 位图形验证码"),
      trigger: "blur"
    }
  ],
  code: [
    {
      required: true,
      pattern: /^\d{6}$/,
      message: tr("请输入 6 位邮箱验证码"),
      trigger: "blur"
    }
  ]
}));
let request = 0,
  timer: ReturnType<typeof setInterval>;
const seconds = ref(0);
async function fetchCaptcha() {
  const id = ++request;
  captchaLoading.value = true;
  captchaFailed.value = false;
  captchaImage.value = "";
  captchaId.value = "";
  form.image = "";
  try {
    const res = await getCaptcha();
    if (id !== request) return;
    if (res?.code !== 200 || !res?.data?.image || !res?.data?.idKey)
      throw new Error();
    captchaImage.value = res.data.image;
    captchaId.value = res.data.idKey;
  } catch {
    if (id === request) captchaFailed.value = true;
  } finally {
    if (id === request) captchaLoading.value = false;
  }
}
function changeMode(value: typeof mode.value) {
  if (loading.value) return;
  mode.value = value;
  verificationId.value = "";
  success.value = "";
  sentEmail.value = "";
  Object.assign(form, {
    username: "",
    email: "",
    password: "",
    confirm: "",
    image: "",
    code: ""
  });
  formRef.value?.clearValidate();
  fetchCaptcha();
}
function retryCode() {
  verificationId.value = "";
  form.code = "";
  form.password = "";
  form.confirm = "";
  fetchCaptcha();
}
async function submit() {
  if (loading.value || !(await formRef.value?.validate().catch(() => false)))
    return;
  if (!stepTwo.value && (!captchaId.value || captchaLoading.value)) return;
  loading.value = true;
  try {
    if (mode.value === "login") {
      const res = await useUserStoreHook().loginByUsername({
        username: form.username,
        password: form.password,
        image: form.image,
        idKey: captchaId.value
      });
      if (res.code !== 200) throw new Error(res.message || tr("登录失败"));
      usePermissionStoreHook().handleWholeMenus([]);
      await router.push("/dashboard");
    } else if (!stepTwo.value) {
      if (!options.emailEnabled)
        throw new Error(tr("邮箱验证尚未配置，请联系管理员"));
      const res = await sendAccountCode({
        email: form.email,
        purpose: mode.value,
        image: form.image,
        idKey: captchaId.value
      });
      if (res.code !== 200) throw new Error(res.message);
      verificationId.value = res.data.verificationId;
      sentEmail.value = form.email;
      seconds.value = res.data.retryAfter || 60;
      formRef.value?.clearValidate();
    } else {
      const payload = {
        verificationId: verificationId.value,
        email: sentEmail.value,
        username: form.username,
        password: form.password,
        code: form.code
      };
      const res = await (mode.value === "register"
        ? registerAccount(payload)
        : resetAccountPassword(payload));
      if (res.code !== 200) throw new Error(res.message);
      success.value =
        mode.value === "register"
          ? "注册成功，请等待管理员启用账号并分配权限。"
          : "密码已重置，原有登录会话已失效，请使用新密码登录。";
      form.password = "";
      form.confirm = "";
      form.code = "";
    }
  } catch (e: any) {
    message(tr(e?.message || "请求失败，请稍后重试"), { type: "error" });
    if (!stepTwo.value) await fetchCaptcha();
  } finally {
    loading.value = false;
  }
}
onMounted(async () => {
  fetchCaptcha();
  timer = setInterval(() => {
    if (seconds.value > 0) seconds.value--;
  }, 1000);
  try {
    const res = await getAccountOptions();
    if (res.code === 200) Object.assign(options, res.data, { loaded: true });
  } catch {
    /* Login remains available. */
  }
});
onBeforeUnmount(() => {
  request++;
  clearInterval(timer);
});
</script>
<template>
  <main class="auth-page">
    <header class="auth-topbar">
      <a
        class="auth-brand"
        href="https://github.com/zhou2004/AutoOps"
        target="_blank"
        rel="noopener noreferrer"
        ><BrandLogo :size="38" /><span
          >AutoOps<small>OPERATIONS, CONNECTED.</small></span
        ></a
      >
      <div class="auth-top-actions">
        <a
          class="auth-docs"
          href="https://github.com/zhou2004/AutoOps#快速开始"
          target="_blank"
          rel="noopener noreferrer"
          >{{ tr("使用文档") }} ↗</a
        ><LanguageSwitch /><el-button
          circle
          :aria-label="tr(dataTheme ? '切换到亮色主题' : '切换到暗色主题')"
          @click="
            dataTheme = !dataTheme;
            dataThemeChange(dataTheme ? 'dark' : 'light');
          "
          ><el-icon><Sunny v-if="dataTheme" /><Moon v-else /></el-icon
        ></el-button>
      </div>
    </header>
    <div class="auth-main">
      <section class="auth-story" :aria-label="tr('平台概览')">
        <span class="auth-kicker"><i />{{ tr("让运维，井然有序") }}</span>
        <h1>
          {{ tr("连接你的基础设施") }}<br /><em>{{ tr("掌控每一次变更") }}</em>
        </h1>
        <p class="auth-intro">
          {{
            tr(
              "从资产到应用，从日常巡检到自动化交付。让复杂的运维工作，回归清晰、可控。"
            )
          }}
        </p>
        <div class="auth-network" aria-hidden="true">
          <div class="network-grid" />
          <svg class="network-lines" viewBox="0 0 600 280">
            <path
              d="M100 70H205Q230 70 230 95V140H300M500 70H395Q370 70 370 95V140H300M100 220H205Q230 220 230 195V140H300M500 220H395Q370 220 370 195V140H300"
            />
            <path
              class="network-pulse"
              d="M100 70H205Q230 70 230 95V140H300H370V195Q370 220 395 220H500"
            />
          </svg>
          <div class="network-core">
            <BrandLogo :size="72" /><strong>AutoOps</strong
            ><span>{{ tr("统一运维工作空间") }}</span>
          </div>
          <div class="network-node node-assets">
            <span class="node-symbol">▤</span
            ><strong>{{ tr("资产管理") }}</strong
            ><small>CMDB</small>
          </div>
          <div class="network-node node-clusters">
            <span class="node-symbol">⬡</span
            ><strong>{{ tr("容器与集群") }}</strong
            ><small>KUBERNETES</small>
          </div>
          <div class="network-node node-monitor">
            <span class="node-symbol">⌁</span
            ><strong>{{ tr("可观测性") }}</strong
            ><small>MONITORING</small>
          </div>
          <div class="network-node node-tasks">
            <span class="node-symbol">↗</span
            ><strong>{{ tr("自动化交付") }}</strong
            ><small>AUTOMATION</small>
          </div>
        </div>
        <div class="auth-capabilities">
          <span><i />{{ tr("统一资产视图") }}</span
          ><span><i />{{ tr("精细权限控制") }}</span
          ><span><i />{{ tr("操作全程留痕") }}</span>
        </div>
      </section>
      <section class="auth-card" :aria-label="title">
        <div v-if="success" class="auth-success">
          <el-icon><Check /></el-icon>
          <h2>{{ tr("验证完成") }}</h2>
          <p role="status">{{ tr(success) }}</p>
          <el-button type="primary" @click="changeMode('login')">{{
            tr("返回登录")
          }}</el-button>
        </div>
        <template v-else>
          <div class="auth-card-heading">
            <span class="auth-step">{{
              mode === "login"
                ? "WORKSPACE ACCESS"
                : stepTwo
                  ? "02 / VERIFY"
                  : "01 / EMAIL"
            }}</span>
            <h2>{{ title }}</h2>
            <p>{{ caption }}</p>
          </div>
          <div
            v-if="mode !== 'reset'"
            class="auth-tabs"
            role="tablist"
            :aria-label="tr('账号操作')"
          >
            <button
              role="tab"
              :aria-selected="mode === 'login'"
              :class="{ active: mode === 'login' }"
              @click="changeMode('login')"
            >
              {{ tr("登录") }}</button
            ><button
              v-if="options.registrationEnabled"
              role="tab"
              :aria-selected="mode === 'register'"
              :class="{ active: mode === 'register' }"
              @click="changeMode('register')"
            >
              {{ tr("注册账号") }}
            </button>
          </div>
          <el-alert
            v-if="mode !== 'login' && !options.emailEnabled"
            :title="tr('邮箱验证尚未配置，请联系管理员')"
            type="info"
            :closable="false"
            show-icon
          />
          <p v-if="stepTwo" class="auth-email-hint">
            {{ tr("如邮箱符合条件，验证码已发送至") }}
            <strong>{{ sentEmail }}</strong>
          </p>
          <el-form
            ref="formRef"
            :model="form"
            :rules="rules"
            label-position="top"
            size="large"
            @submit.prevent="submit"
          >
            <el-form-item
              v-if="mode === 'login' || (mode === 'register' && stepTwo)"
              prop="username"
              :label="tr('账号')"
              ><el-input
                v-model="form.username"
                autocomplete="username"
                :placeholder="tr('请输入账号')"
                maxlength="32"
            /></el-form-item>
            <el-form-item
              v-if="mode !== 'login' && !stepTwo"
              prop="email"
              :label="tr('邮箱')"
              ><el-input
                v-model="form.email"
                type="email"
                autocomplete="email"
                placeholder="you@company.com"
                maxlength="254"
            /></el-form-item>
            <el-form-item v-if="stepTwo" prop="code" :label="tr('邮箱验证码')"
              ><el-input
                v-model="form.code"
                inputmode="numeric"
                autocomplete="one-time-code"
                maxlength="6"
                :placeholder="tr('请输入 6 位邮箱验证码')"
            /></el-form-item>
            <el-form-item
              v-if="mode === 'login' || stepTwo"
              prop="password"
              :label="tr(mode === 'reset' ? '新密码' : '密码')"
              ><el-input
                v-model="form.password"
                type="password"
                show-password
                :autocomplete="
                  mode === 'login' ? 'current-password' : 'new-password'
                "
                :placeholder="
                  tr(
                    mode === 'login'
                      ? '请输入密码'
                      : '至少 10 位，建议使用长密码'
                  )
                "
            /></el-form-item>
            <el-form-item v-if="stepTwo" prop="confirm" :label="tr('确认密码')"
              ><el-input
                v-model="form.confirm"
                type="password"
                show-password
                autocomplete="new-password"
                :placeholder="tr('再次输入密码')"
            /></el-form-item>
            <el-form-item v-if="!stepTwo" prop="image" :label="tr('图形验证码')"
              ><div class="auth-captcha-row">
                <el-input
                  v-model="form.image"
                  inputmode="numeric"
                  autocomplete="off"
                  maxlength="4"
                  :placeholder="tr('4 位数字')"
                /><button
                  class="auth-captcha"
                  type="button"
                  :disabled="captchaLoading"
                  :aria-label="tr('点击刷新验证码')"
                  @click="fetchCaptcha"
                >
                  <img
                    v-if="captchaImage"
                    :src="captchaImage"
                    :alt="tr('图形验证码')"
                    width="200"
                    height="60"
                    @error="
                      captchaImage = '';
                      captchaFailed = true;
                    "
                  /><span v-else>{{
                    tr(captchaFailed ? "点击重试" : "加载中…")
                  }}</span></button
                ><button
                  class="auth-refresh"
                  type="button"
                  :aria-label="tr('刷新验证码')"
                  :disabled="captchaLoading"
                  @click="fetchCaptcha"
                >
                  <el-icon><Refresh /></el-icon>
                </button></div
            ></el-form-item>
            <div v-if="mode === 'login'" class="auth-form-meta">
              <span>{{ tr("安全访问你的工作空间") }}</span
              ><button type="button" @click="changeMode('reset')">
                {{ tr("忘记密码？") }}
              </button>
            </div>
            <el-button
              class="auth-submit"
              type="primary"
              native-type="submit"
              :loading="loading"
              :disabled="
                (mode !== 'login' && !options.emailEnabled) ||
                (!stepTwo && (captchaLoading || !captchaId))
              "
              >{{
                tr(
                  mode === "login"
                    ? "进入工作空间"
                    : !stepTwo
                      ? "发送邮箱验证码"
                      : mode === "register"
                        ? "完成注册"
                        : "重置密码"
                )
              }}<el-icon><ArrowRight /></el-icon
            ></el-button>
          </el-form>
          <div v-if="mode !== 'login'" class="auth-return">
            <button type="button" @click="changeMode('login')">
              <el-icon><ArrowLeft /></el-icon>{{ tr("返回登录") }}</button
            ><button
              v-if="stepTwo"
              type="button"
              :disabled="seconds > 0"
              @click="retryCode"
            >
              {{
                seconds > 0
                  ? tr("{seconds} 秒后可重新发送", { seconds })
                  : tr("重新获取验证码")
              }}
            </button>
          </div>
          <p class="auth-card-note">
            {{
              tr(
                mode === "register"
                  ? "注册后需管理员启用并授权，才可访问平台资源。"
                  : mode === "reset"
                    ? "请使用账号已绑定的邮箱；未绑定邮箱请联系管理员。"
                    : "仅授权用户可访问，操作将按平台配置记录审计。"
              )
            }}
          </p>
        </template>
      </section>
    </div>
    <footer class="auth-footer">
      <span>AutoOps · {{ tr("让每一次运维更从容") }}</span
      ><a
        href="https://github.com/zhou2004/AutoOps"
        target="_blank"
        rel="noopener noreferrer"
        >GitHub ↗</a
      >
    </footer>
  </main>
</template>
