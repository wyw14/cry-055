export type InstrumentStatus='pending'|'qualified'|'due_soon'|'overdue'|'unqualified'|'disabled'|'reinspection'
export interface Instrument{id:string;asset_number:string;name:string;model:string;location:string;criticality:'normal'|'major'|'critical';status:InstrumentStatus;next_due_at:string;version:number}
export interface Page<T>{items:T[];page:number;size:number;total:number}
export interface ApiError{code:string;message:string;field_errors:{field:string;message:string}[];request_id:string}
export interface Alert{id:string;severity:'info'|'warning'|'critical';kind:string;title:string;message:string;created_at:string}
export interface CalendarEntry{id:string;assetNumber:string;name:string;date:string;status:InstrumentStatus}
