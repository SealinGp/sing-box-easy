// 个人中心页面汉化 (Profile.vue)。
export default {
  title: '个人中心与账号管理',
  subtitle: '管理您的账号凭证和系统操作员',
  tabs: {
    profile: '我的资料',
    users: '账号管理',
  },
  info: {
    createdVal: '创建时间',
  },
  roles: {
    admin: '管理员',
    viewer: '只读用户',
  },
  profileSection: {
    title: '修改凭证',
    username: '用户名',
    newPassword: '新密码（留空表示不修改）',
    confirmPassword: '确认新密码',
    saveBtn: '保存修改',
  },
  usersSection: {
    title: '操作员账号',
    addBtn: '添加用户',
    table: {
      id: 'ID',
      username: '用户名',
      role: '角色',
      created: '创建时间',
      actions: '操作',
    },
  },
  modals: {
    addTitle: '添加操作员账号',
    editTitle: '编辑用户',
    username: '用户名',
    password: '密码',
    role: '角色',
    roleViewerDesc: '只读用户 (仅查看日志/状态)',
    roleAdminDesc: '管理员 (完全控制)',
    resetPassword: '重置密码（留空表示不修改）',
  },
  validation: {
    passwordMismatch: '两次输入的密码不一致',
    requiredFields: '用户名和密码不能为空',
  },
  toast: {
    loadProfileFailed: '加载个人资料失败',
    loadUsersFailed: '获取用户列表失败',
    profileUpdated: '个人资料已成功更新',
    userCreated: "用户 '{username}' 已成功创建",
    createUserFailed: '创建用户失败',
    userUpdated: "用户 '{username}' 已成功更新",
    updateUserFailed: '更新用户失败',
    deleteUserConfirm: "确定要删除用户 '{username}' 吗？",
    userDeleted: "用户 '{username}' 已成功删除",
    deleteUserFailed: '删除用户失败',
  },
}
