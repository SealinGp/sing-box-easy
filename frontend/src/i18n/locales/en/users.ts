// User Management localization keys (Users.vue).
export default {
  title: 'User Management',
  add: 'Add User',
  addFirst: 'Add your first user',
  empty: 'No users found',
  table: {
    username: 'Username',
    role: 'Role',
    actions: 'Actions',
  },
  modal: {
    add: 'Add New User',
    edit: 'Edit User',
  },
  form: {
    username: 'Username',
    usernamePlaceholder: 'Enter username',
    password: 'Password',
    passwordPlaceholder: 'Enter password (leave empty to keep unchanged)',
    passwordRequiredPlaceholder: 'Enter password',
    role: 'Role',
    rolePlaceholder: 'Select role',
    roles: {
      admin: 'Administrator',
      viewer: 'Viewer',
    },
  },
  validation: {
    title: 'Validation Error',
    usernameRequired: 'Username is required',
    passwordRequired: 'Password is required',
  },
  toast: {
    fetchFailed: 'Failed to fetch users',
    addedOk: 'User added successfully',
    updatedOk: 'User updated successfully',
    saveFailed: 'Failed to save user',
    deletedOk: 'User deleted successfully',
    deleteFailed: 'Failed to delete user',
  },
  del: {
    title: 'Delete User',
    confirm: 'Are you sure you want to delete user "{username}"? This action cannot be undone.',
  },
}
