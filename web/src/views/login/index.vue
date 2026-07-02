<script setup lang="ts">
import Motion from "./utils/motion";
import { useRouter } from "vue-router";
import { message } from "@/utils/message";
import { loginRules } from "./utils/rule";
import { ref, reactive, toRaw, onMounted } from "vue";
import { debounce } from "@pureadmin/utils";
import { useNav } from "@/layout/hooks/useNav";
import { useEventListener } from "@vueuse/core";
import type { FormInstance } from "element-plus";
import { useLayout } from "@/layout/hooks/useLayout";
import { useUserStoreHook } from "@/store/modules/user";
import { usePermissionStoreHook } from "@/store/modules/permission";
import { bg, avatar, illustration } from "./utils/static";
import { useRenderIcon } from "@/components/ReIcon/src/hooks";
import { useDataThemeChange } from "@/layout/hooks/useDataThemeChange";
import { getCaptcha } from "@/api/user";

import dayIcon from "@/assets/svg/day.svg?component";
import darkIcon from "@/assets/svg/dark.svg?component";
import Lock from "~icons/ri/lock-fill";
import User from "~icons/ri/user-3-fill";
import PictureFilled from "~icons/ep/picture-filled";

defineOptions({
  name: "Login"
});

const router = useRouter();
const loading = ref(false);
const disabled = ref(false);
const ruleFormRef = ref<FormInstance>();
const captchaImage = ref("");       // base64 验证码图片
const captchaIdKey = ref("");       // 验证码 idKey

const { initStorage } = useLayout();
initStorage();

const { dataTheme, overallStyle, dataThemeChange } = useDataThemeChange();
dataThemeChange(overallStyle.value);
const { title } = useNav();

const ruleForm = reactive({
  username: "admin",
  password: "123456",
  image: ""    // 验证码输入
});

/** 获取验证码 */
const fetchCaptcha = async () => {
  try {
    const res: any = await getCaptcha();
    if (res?.code === 200 && res?.data) {
      captchaImage.value = res.data.image;
      captchaIdKey.value = res.data.idKey;
    } else {
      message(res?.message || "获取验证码失败", { type: "error" });
    }
  } catch (err) {
    message("获取验证码失败，请检查网络连接", { type: "error" });
  }
};

onMounted(() => {
  fetchCaptcha();
});

const onLogin = async (formEl: FormInstance | undefined) => {
  if (!formEl) return;
  await formEl.validate(valid => {
    if (valid) {
      if (!captchaIdKey.value) {
        message("请先获取验证码", { type: "warning" });
        return;
      }
      loading.value = true;
      useUserStoreHook()
        .loginByUsername({
          username: ruleForm.username,
          password: ruleForm.password,
          image: ruleForm.image,
          idKey: captchaIdKey.value
        })
        .then((res: any) => {
          if (res?.code === 200) {
            // 从后端返回的数据构建菜单并注入权限 store
            usePermissionStoreHook().handleWholeMenus([]);
            disabled.value = true;
            router
              .push("/dashboard")
              .then(() => {
                message("登录成功", { type: "success" });
              })
              .finally(() => (disabled.value = false));
          } else {
            message(res?.message || "登录失败", { type: "error" });
            // 登录失败刷新验证码
            fetchCaptcha();
            ruleForm.image = "";
          }
        })
        .catch(() => {
          message("登录请求失败", { type: "error" });
          fetchCaptcha();
        })
        .finally(() => (loading.value = false));
    }
  });
};

const immediateDebounce: any = debounce(
  formRef => onLogin(formRef),
  1000,
  true
);

useEventListener(document, "keydown", ({ code }) => {
  if (
    ["Enter", "NumpadEnter"].includes(code) &&
    !disabled.value &&
    !loading.value
  )
    immediateDebounce(ruleFormRef.value);
});
</script>

<template>
  <div class="select-none">
    <img :src="bg" class="wave" />
    <div class="flex-c absolute right-5 top-3">
      <!-- 主题 -->
      <el-switch
        v-model="dataTheme"
        inline-prompt
        :active-icon="dayIcon"
        :inactive-icon="darkIcon"
        @change="dataThemeChange"
      />
    </div>
    <div class="login-container">
      <div class="img">
        <component :is="toRaw(illustration)" />
      </div>
      <div class="login-box">
        <div class="login-form">
          <avatar class="avatar" />
          <Motion>
            <h2 class="outline-hidden">AutoOps</h2>
          </Motion>

          <el-form
            ref="ruleFormRef"
            :model="ruleForm"
            :rules="loginRules"
            size="large"
          >
            <Motion :delay="100">
              <el-form-item
                :rules="[
                  {
                    required: true,
                    message: '请输入账号',
                    trigger: 'blur'
                  }
                ]"
                prop="username"
              >
                <el-input
                  v-model="ruleForm.username"
                  clearable
                  placeholder="账号"
                  :prefix-icon="useRenderIcon(User)"
                />
              </el-form-item>
            </Motion>

            <Motion :delay="150">
              <el-form-item prop="password">
                <el-input
                  v-model="ruleForm.password"
                  clearable
                  show-password
                  placeholder="密码"
                  :prefix-icon="useRenderIcon(Lock)"
                />
              </el-form-item>
            </Motion>

            <!-- 验证码 -->
            <Motion :delay="200">
              <el-form-item prop="image">
                <div class="captcha-row">
                  <el-input
                    v-model="ruleForm.image"
                    clearable
                    placeholder="验证码"
                    maxlength="6"
                    :prefix-icon="useRenderIcon(PictureFilled)"
                    style="flex: 1"
                  />
                  <div class="captcha-box" @click="fetchCaptcha" title="点击刷新验证码">
                    <img v-if="captchaImage" :src="captchaImage" class="captcha-img" alt="验证码" />
                    <span v-else class="captcha-placeholder">加载中...</span>
                  </div>
                </div>
              </el-form-item>
            </Motion>

            <Motion :delay="250">
              <el-button
                class="w-full mt-4!"
                size="default"
                type="primary"
                :loading="loading"
                :disabled="disabled"
                @click="onLogin(ruleFormRef)"
              >
                登录
              </el-button>
            </Motion>
          </el-form>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
@import url("@/style/login.css");

.captcha-row {
  display: flex;
  align-items: center;
  gap: 8px;
}
.captcha-box {
  flex-shrink: 0;
  width: 120px;
  height: 40px;
  cursor: pointer;
  border-radius: 4px;
  overflow: hidden;
  border: 1px solid var(--el-border-color);
  display: flex;
  align-items: center;
  justify-content: center;
}
.captcha-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.captcha-placeholder {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
</style>

<style lang="scss" scoped>
:deep(.el-input-group__append, .el-input-group__prepend) {
  padding: 0;
}
</style>
