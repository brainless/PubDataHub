import axios from 'axios';
import { HomeResponse, ErrorResponse } from '../types/api';

const API_BASE_URL = 'http://localhost:8080';

const api = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
});

export const homeAPI = {
  getHomeDirectory: async (): Promise<HomeResponse> => {
    try {
      const response = await api.get<HomeResponse>('/api/home');
      return response.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response?.data) {
        const errorData = error.response.data as ErrorResponse;
        throw new Error(errorData.message || 'Failed to fetch home directory');
      }
      throw new Error('Network error occurred');
    }
  },
};

export default api;