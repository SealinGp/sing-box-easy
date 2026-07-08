import { ApiService } from './api'
import type { User, LoginResponse, BasicResponse } from '../types/api'

export class UserService {
  private api: ApiService

  constructor(api: ApiService) {
    this.api = api
  }

  async login(username: string, password: string): Promise<LoginResponse> {
    const { data } = await this.api.post<BasicResponse<LoginResponse>>('/user/login', { username, password })
    this.api.setToken(data.data.token)
    return data.data
  }

  async logout(): Promise<void> {
    try {
      await this.api.post<BasicResponse<void>>('/user/logout')
    } finally {
      this.api.clearToken()
    }
  }

  async getMe(): Promise<User> {
    const { data } = await this.api.get<BasicResponse<User>>('/user/me')
    return data.data
  }

  async listUsers(): Promise<User[]> {
    const { data } = await this.api.get<BasicResponse<User[]>>('/users')
    return data.data
  }

  async createUser(user: Partial<User> & { password?: string }): Promise<User> {
    const { data } = await this.api.post<BasicResponse<User>>('/users', user)
    return data.data
  }

  async updateUser(id: number, user: Partial<User> & { password?: string }): Promise<User> {
    const { data } = await this.api.put<BasicResponse<User>>(`/users/${id}`, user)
    return data.data
  }

  async deleteUser(id: number): Promise<void> {
    await this.api.delete<BasicResponse<void>>(`/users/${id}`)
  }
}
