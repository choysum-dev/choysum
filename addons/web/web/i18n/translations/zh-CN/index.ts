// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { TranslationMessages } from '../../source';

/**
 * Simplified Chinese locale pack translated from the English source.
 * In production projects, this is typically exported from a translation tool.
 */
const messages: TranslationMessages = {
  common: {
    // Buttons and actions.
    confirm: '确认',
    cancel: '取消',
    save: '保存',
    delete: '删除',
    edit: '编辑',
    view: '查看',
    search: '搜索',
    filter: '筛选',
    reset: '重置',
    more: '更多',
    actions: '操作',
    submit: '提交',
    clear: '清除',
    close: '关闭',

    // State and feedback.
    loading: '加载中...',
    noData: '暂无数据',
    noResults: '未找到结果',
    required: '必填',
    optional: '可选',
    enabled: '已启用',
    disabled: '已禁用',
    yes: '是',
    no: '否',

    // Navigation.
    back: '返回',
    next: '下一步',
    previous: '上一步',

    // User labels.
    welcome: '欢迎',
    login: '登录',
    logout: '退出登录',
    register: '注册',

    // Common features.
    upload: '上传',
    download: '下载',
    refresh: '刷新',
    settings: '设置',
    help: '帮助',
  },

  layout: {
    appName: 'Choysum',

    sidebar: {
      collapse: '收起侧边栏',
      expand: '展开侧边栏',
      toggle: '切换侧边栏',
      home: '首页',
      dashboard: '仪表盘',
      menu: '菜单',
    },

    header: {
      menu: '菜单',
      profile: '个人资料',
      settings: '设置',
      notifications: '通知',
      languages: '语言',
      darkMode: '深色模式',
      lightMode: '浅色模式',
      search: '搜索...',
    },

    footer: {
      copyright: '© {year} Choysum. 保留所有权利。',
      version: '版本 {version}',
      powered: '由 Choysum 提供支持',
    },

    breadcrumb: {
      home: '首页',
    },

    page: {
      loading: '正在加载页面内容...',
      notFound: '页面未找到',
      accessDenied: '访问被拒绝',
    },
  },

  form: {
    validation: {
      required: '此字段为必填项',
      email: '请输入有效的电子邮件地址',
      url: '请输入有效的URL',
      number: '请输入有效的数字',
      integer: '请输入有效的整数',
      minLength: '请至少输入 {min} 个字符',
      maxLength: '请不要超过 {max} 个字符',
      minValue: '请输入不小于 {min} 的值',
      maxValue: '请输入不大于 {max} 的值',
      pattern: '请输入符合要求格式的值',
      passwordMatch: '密码不匹配',
    },

    placeholder: {
      search: '搜索...',
      select: '请选择',
      date: '选择日期',
      time: '选择时间',
      datetime: '选择日期和时间',
      enterText: '请输入内容...',
    },

    button: {
      submit: '提交',
      reset: '重置',
      cancel: '取消',
      search: '搜索',
      filter: '筛选',
    },

    label: {
      username: '用户名',
      password: '密码',
      confirmPassword: '确认密码',
      email: '电子邮件',
      phone: '电话',
      address: '地址',
      dateOfBirth: '出生日期',
      gender: '性别',
      firstName: '名',
      lastName: '姓',
    },
  },
};

export default messages;
