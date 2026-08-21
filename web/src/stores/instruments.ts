import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { api } from '../services/api'
import type { Instrument, InstrumentStatus } from '../types'

export const useInstrumentStore=defineStore('instruments',()=>{const instruments=ref<Instrument[]>([]);const total=ref(0);const loading=ref(false);const error=ref('');const status=ref<InstrumentStatus|''>('');const query=ref('');const blockedCount=computed(()=>instruments.value.filter(item=>['overdue','unqualified','disabled'].includes(item.status)).length);async function load(){loading.value=true;error.value='';try{const params=new URLSearchParams({page:'1',size:'100',sort:'next_due_at'});if(status.value)params.set('filter[status]',status.value);const result=await api.listInstruments(params);instruments.value=result.items.filter(item=>!query.value||`${item.asset_number}${item.name}${item.model}`.toLowerCase().includes(query.value.toLowerCase()));total.value=result.total}catch(reason){error.value=reason instanceof Error?reason.message:'加载失败'}finally{loading.value=false}}return{instruments,total,loading,error,status,query,blockedCount,load}})
