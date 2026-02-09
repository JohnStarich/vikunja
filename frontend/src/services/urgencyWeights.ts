import AbstractService from './abstractService'
import UserUrgencyWeightsModel from '@/models/userUrgencyWeights'
import type {IUserUrgencyWeights} from '@/modelTypes/IUserUrgencyWeights'

export default class UserUrgencyWeightsService extends AbstractService<IUserUrgencyWeights> {
	constructor() {
		super({
			get: '/user/settings/urgency_weights',
			update: '/user/settings/urgency_weights',
		})
	}

	modelFactory(data: Partial<IUserUrgencyWeights>) {
		return new UserUrgencyWeightsModel(data)
	}
}
