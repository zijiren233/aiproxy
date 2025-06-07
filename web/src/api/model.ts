// src/api/model.ts
import { get, post, del, put } from './index'
import { ModelConfig, ModelCreateRequest } from '@/types/model'


export const modelApi = {
    getModels: async (): Promise<ModelConfig[]> => {
        const response = await get<ModelConfig[]>('model_configs/all')
        return response
    },

    getModel: async (model: string): Promise<ModelConfig> => {
        const response = await get<ModelConfig>(`model_config/${model}`)
        return response
    },

    createModel: async (data: ModelCreateRequest): Promise<void> => {
        await post('model_config/', data)
        return
    },

    updateModel: async (model: string, data: Omit<ModelCreateRequest, 'model'>): Promise<void> => {
        await put(`model_config/${model}`, data)
        return
    },

    deleteModel: async (model: string): Promise<void> => {
        await del(`model_config/${model}`)
        return
    }
}