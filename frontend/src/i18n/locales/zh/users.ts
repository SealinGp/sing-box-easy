// 用户管理页面汉化 (Users.vue)。
export default {
  title: '用户管理',
  add: '添加用户',
  addFirst: '添加第一个用户',
  empty: '未找到用户',
  table: {
    username: '用户名',
    role: '角色',
    actions: '操作',
  },
  modal: {
    add: '添加新用户',
    edit: '编辑用户',
  },
  form: {
    username: '用户名',
    usernamePlaceholder: '请输入用户名',
    password: '密码',
    passwordPlaceholder: '请输入密码（留空表示不修改）',
    passwordRequiredPlaceholder: '请输入密码',
    role: '角色',
    rolePlaceholder: '请选择角色',
    roles: {
      admin: '管理员',
      viewer: '只读用户',
    },
  },
  validation: {
    title: '验证错误',
    usernameRequired: '用户名不能为空',
    passwordRequired: '密码不能为空',
  },
  toast: {
    fetchFailed: '获取用户列表失败',
    addedOk: '用户添加成功',
    updatedOk: '用户更新成功',
    saveFailed: '保存用户失败',
    deletedOk: '用户删除成功',
    deleteFailed: '删除用户失败',
  },
  del: {
    title: '删除用户',
    confirm: '确定要删除用户 "{username}" 吗？此操作无法撤销。',
  },
}
