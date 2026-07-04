<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OPage :loading="loading" class="register-page-container">
    <el-card class="register-card" shadow="hover">
      <template #header>
        <div class="card-header">
          <h3>Create Account</h3>
        </div>
      </template>
      <transition name="fade">
        <el-alert v-if="error" :title="error" type="error" :closable="true" @close="error = ''" />
      </transition>

      <el-form ref="registerForm" :model="form" :rules="rules" label-position="top" @keydown="handleKeyDown" @submit.prevent="handleRegister">
        <el-form-item prop="username" label="Username">
          <el-input v-model="form.username" placeholder="Enter username" :prefix-icon="User" autocomplete="username" />
        </el-form-item>

        <el-form-item prop="email" label="Email">
          <el-input v-model="form.email" placeholder="Enter email address" :prefix-icon="Message" type="email" autocomplete="email" />
        </el-form-item>

        <el-form-item prop="password" label="Password">
          <el-input v-model="form.password" placeholder="Enter password" :prefix-icon="Lock" type="password" autocomplete="new-password" show-password />
        </el-form-item>

        <el-form-item prop="confirmPassword" label="Confirm Password">
          <el-input
            v-model="form.confirmPassword"
            placeholder="Re-enter password"
            :prefix-icon="Lock"
            type="password"
            autocomplete="new-password"
            show-password
          />
        </el-form-item>

        <el-form-item prop="agreeTerms">
          <el-checkbox v-model="form.agreeTerms">
            I have read and agree to <a href="#" target="_blank">Terms of Service</a> and
            <a href="#" target="_blank">Privacy Policy</a>
          </el-checkbox>
        </el-form-item>

        <el-form-item>
          <el-button type="primary" native-type="submit" :loading="loading" class="submit-button" :disabled="!form.agreeTerms"> Create Account </el-button>
        </el-form-item>

        <div class="login-link">
          Already have an account?
          <router-link to="/login">Log in now</router-link>
        </div>
      </el-form>
    </el-card>
  </OPage>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue';
import { useRouter, useRoute } from 'vue-router';
import { storeToRefs } from 'pinia';
import { useAuthStore } from '../stores/auth';
import { ChoysumError } from '../error';
import { ElForm, ElFormItem, ElInput, ElButton, ElAlert, ElCheckbox, ElCard } from 'element-plus';
import { User, Lock, Message } from '@element-plus/icons-vue';
import OPage from '@/web/web/components/page/OPage.vue';

const router = useRouter();
const route = useRoute();
const authStore = useAuthStore();
const { loading } = storeToRefs(authStore);

const registerForm = ref();
const form = reactive({
  username: '',
  email: '',
  password: '',
  confirmPassword: '',
  fullName: '',
  agreeTerms: false,
});

const error = ref('');

/**
 * Validate the username field against local registration rules.
 */
const validateUsername = (rule: any, value: string, callback: any) => {
  if (!value) {
    callback(new Error('Enter username'));
  } else if (value.length < 3) {
    callback(new Error('Username must be at least 3 characters'));
  } else if (!/^[a-zA-Z0-9_\-\.]+$/.test(value)) {
    callback(new Error('Username can only contain letters, numbers, underscores, hyphens, and dots'));
  } else {
    callback();
  }
};

/**
 * Validate the email field against the expected address format.
 */
const validateEmail = (rule: any, value: string, callback: any) => {
  if (!value) {
    callback(new Error('Enter email address'));
  } else if (!/^[\w-]+(\.[\w-]+)*@[\w-]+(\.[\w-]+)+$/.test(value)) {
    callback(new Error('Enter a valid email address'));
  } else {
    callback();
  }
};

/**
 * Validate the password field and recheck confirmation when needed.
 */
const validatePassword = (rule: any, value: string, callback: any) => {
  if (!value) {
    callback(new Error('Enter password'));
  } else if (value.length < 6) {
    callback(new Error('Password must be at least 6 characters'));
  } else {
    // Revalidate the confirmation field after the password changes.
    if (form.confirmPassword) {
      registerForm.value.validateField('confirmPassword');
    }
    callback();
  }
};

/**
 * Validate that the confirmation password matches the primary password.
 */
const validateConfirmPassword = (rule: any, value: string, callback: any) => {
  if (!value) {
    callback(new Error('Re-enter password'));
  } else if (value !== form.password) {
    callback(new Error('Passwords do not match'));
  } else {
    callback();
  }
};

/**
 * Validate that the user accepted the required terms.
 */
const validateAgreeTerms = (rule: any, value: boolean, callback: any) => {
  if (!value) {
    callback(new Error('You must agree to the Terms of Service and Privacy Policy'));
  } else {
    callback();
  }
};

const rules = {
  username: [{ validator: validateUsername, trigger: 'blur' }],
  email: [{ validator: validateEmail, trigger: 'blur' }],
  password: [{ validator: validatePassword, trigger: 'blur' }],
  confirmPassword: [{ validator: validateConfirmPassword, trigger: 'blur' }],
  fullName: [{ required: false, message: 'Enter your name', trigger: 'blur' }],
  agreeTerms: [{ validator: validateAgreeTerms, trigger: 'change' }],
};

/**
 * Validate the registration form and create a new user session.
 */
async function handleRegister() {
  if (!registerForm.value) return;

  try {
    await registerForm.value.validate();

    error.value = '';

    await authStore.register(form.username, form.email, form.password, form.fullName ? { fullName: form.fullName } : {});

    await authStore.login(form.username, form.password);

    const redirect = route.query.redirect?.toString() || '/';
    router.replace(redirect);
  } catch (err) {
    if (err instanceof ChoysumError) {
      error.value = err.message;
    } else {
      error.value = 'Registration failed. Please try again later.';
      console.error('Registration flow failed:', err);
    }
  }
}

/**
 * Navigate back to the login page.
 */
function goToLogin() {
  router.push('/login');
}

/**
 * Submit the form when Enter is pressed and terms are accepted.
 */
function handleKeyDown(event: KeyboardEvent) {
  if (event.key === 'Enter' && form.agreeTerms) {
    handleRegister();
  }
}
</script>

<style lang="scss" scoped>
.register-page-container {
  &.o-page--with-padding {
    padding: 0;
  }

  :deep(.o-page__body) {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100%;
    padding: var(--el-padding-large) 0;
  }
}

.register-card {
  width: 100%;
  max-width: 420px;
}

.card-header {
  text-align: center;
  h3 {
    margin: 0;
    font-size: var(--el-font-size-large);
    font-weight: var(--el-font-weight-bold);
  }
}

.submit-button {
  width: 100%;
}

.login-link {
  text-align: center;
  margin-block-start: var(--el-margin-base, 16px);
  font-size: var(--el-font-size-small, 14px);
  color: var(--el-text-color-secondary);

  a {
    color: var(--el-color-primary);
    text-decoration: none;
    margin-inline-start: var(--el-margin-small, 4px);

    &:hover {
      text-decoration: underline;
    }
  }
}

:deep(.el-checkbox__label) {
  a {
    color: var(--el-color-primary);
    text-decoration: none;

    &:hover {
      text-decoration: underline;
    }
  }
}

:deep(.el-alert) {
  margin-block-end: var(--el-margin-medium, 20px);
}
</style>
