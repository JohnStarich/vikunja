import AbstractService from './abstractService'
import SavedFilterUrgencyWeightsModel from '@/models/savedFilterUrgencyWeights'
import type {ISavedFilterUrgencyWeights} from '@/modelTypes/ISavedFilterUrgencyWeights'

export default class SavedFilterUrgencyWeightsService extends AbstractService<ISavedFilterUrgencyWeights> {
	constructor() {
		super({
			get: '/filters/{id}/urgency_weights',
			create: '/filters/{id}/urgency_weights',
		})
	}

	modelFactory(data: Partial<ISavedFilterUrgencyWeights>) {
		return new SavedFilterUrgencyWeightsModel(data)
	}
}
