import AbstractModel from './abstractModel'

import type {ISavedFilterUrgencyWeights} from '@/modelTypes/ISavedFilterUrgencyWeights'

export default class SavedFilterUrgencyWeightsModel extends AbstractModel<ISavedFilterUrgencyWeights> implements ISavedFilterUrgencyWeights {
	urgencyWeights: []

	constructor(data: Partial<ISavedFilterUrgencyWeights> = {}) {
		super()
		this.assignData(data)
	}
}
