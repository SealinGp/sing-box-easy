// Profile page (Profile.vue).
export default {
  title: 'User Profile & Accounts',
  subtitle: 'Manage your account credentials and system operators',
  tabs: {
    profile: 'My Profile',
    users: 'Manage Accounts',
  },
  info: {
    createdVal: 'Created At',
  },
  roles: {
    admin: 'Administrator',
    viewer: 'Viewer',
  },
  profileSection: {
    title: 'Update Credentials',
    username: 'Username',
    newPassword: 'New Password (leave empty to keep current)',
    confirmPassword: 'Confirm New Password',
    saveBtn: 'Save Changes',
  },
  usersSection: {
    title: 'Operator Accounts',
    addBtn: 'Add User',
    table: {
      id: 'ID',
      username: 'Username',
      role: 'Role',
      created: 'Created At',
      actions: 'Actions',
    },
  },
  modals: {
    addTitle: 'Add Operator Account',
    editTitle: 'Edit User',
    username: 'Username',
    password: 'Password',
    role: 'Role',
    roleViewerDesc: 'Viewer (Read-only logs / status)',
    roleAdminDesc: 'Administrator (Full control)',
    resetPassword: 'Reset Password (leave empty to keep current)',
  },
  validation: {
    passwordMismatch: 'Passwords do not match',
    requiredFields: 'Username and password are required',
  },
  toast: {
    loadProfileFailed: 'Failed to load profile',
    loadUsersFailed: 'Failed to list users',
    profileUpdated: 'Profile updated successfully',
    userCreated: "User '{username}' created successfully",
    createUserFailed: 'Failed to create user',
    userUpdated: "User '{username}' updated successfully",
    updateUserFailed: 'Failed to update user',
    deleteUserConfirm: "Are you sure you want to delete user '{username}'?",
    userDeleted: "User '{username}' deleted successfully",
    deleteUserFailed: 'Failed to delete user',
  },
}
