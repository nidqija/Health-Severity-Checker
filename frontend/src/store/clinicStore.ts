import { reactive } from 'vue'
import type { Clinic } from '../places'

export const clinicStore = reactive<{
  clinics: Clinic[]
  confirmedClinic: Clinic | null
  userLat: number | null
  userLng: number | null
}>({
  clinics: [],
  confirmedClinic: null,
  userLat: null,
  userLng: null,
})
