import AbstractModel from './abstractModel'

import type {IUserUrgencyWeights} from '@/modelTypes/IUserUrgencyWeights'

export default class UserUrgencyWeightsModel extends AbstractModel<IUserUrgencyWeights> implements IUserUrgencyWeights {
	urgencyWeights: []

	constructor(data: Partial<IUserUrgencyWeights> = {}) {
		super()
		this.assignData(data)
	}
}
