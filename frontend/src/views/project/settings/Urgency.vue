<template>
	<Modal @close="$router.back()">
		<Card :title="$t('project.urgentTasks.title')" class="urgent-tasks">
			<p>{{ $t('project.urgentTasks.description') }}</p>
			<p>{{ $t('project.urgentTasks.getStarted') }}</p>

			<div class="weight-creator">
				<Dropdown
					#trigger="{ toggleOpen, open }"
					class="property-picker"
				>
					<BaseButton
						variant="secondary"
						@click="toggleOpen"
					>
						<span>{{ newProperty ? $t('project.urgentTasks.properties.' + newProperty) : "New property" }}</span>
						<span
							class="mis-1 dropdown-icon icon is-small"
							:style="{
								transform: open ? 'rotate(180deg)' : 'rotate(0)',
							}"
						>
							<Icon icon="chevron-down" />
						</span>
					</BaseButton>
					<template v-if="open">
						<DropdownItem
							v-for="property in newPropertyNames()"
							@click.stop="() => {
								newProperty = property
								toggleOpen()
							}"
						>
							{{ $t('project.urgentTasks.properties.'+property) }}
						</DropdownItem>
					</template>
				</Dropdown>
				<XButton variant="primary" @click="addWeight()">
					{{ $t('misc.create') }}
				</XButton>
			</div>
			<Modal
				:enabled="filterEditorIndex !== null"
				transition-name="fade"
				:overflow="true"
				variant="hint-modal"
				@close="filterEditorIndex = null"
			>
				<Filters
					v-model="filterEditorWeight"
					class="filter-popup"
					:change-immediately="false"
					show-close
					@close="filterEditorIndex = null"
					@showResults="updateWeightFilter(filterEditorIndex, filterEditorWeight)"
				/>
			</Modal>
			<table
				v-if="weights.length > 0"
				class="table weights"
			>
				<thead>
					<tr>
						<th>{{ $t('project.urgentTasks.property') }}</th>
						<th>{{ $t('project.urgentTasks.filter') }}</th>
						<th>{{ $t('project.urgentTasks.weight') }}</th>
						<th class="has-text-end">
							{{ $t('misc.actions') }}
						</th>
					</tr>
				</thead>
				<tbody>
					<template v-for="(weight, index) in weights">
						<tr>
							<td class="weight-property">
								{{ weight.propertyName() }}
							</td>
							<td class="weight-filter">
								<FilterInput
									disabled
									v-if="weight.filter"
									class="input filter-input"
									v-model="weight.filter.query"
								/>
							</td>
							<td>
								<input
									class="input weight-input"
									:placeholder="$t('project.urgentTasks.weight')"
									type="number"
									:value="weight.weight"
									min="1"
									@input="e => updateWeight(index, {weight: Number(e.target.value)})"
								>
							</td>
							<td>
								<div class="actions">
									<XButton
										v-if="weight.filter"
										icon="pen"
										@click="editWeightFilter(index)"
									/>
									<XButton
										class="is-danger"
										icon="trash-alt"
										@click="deleteWeight(index)"
									/>
								</div>
							</td>
						</tr>
						<tr class="weight-proportion">
							<td
								colspan="4"
								class="weight-proportion-container"
								:title="`${weight.displayName()} is ${Math.round(weight.weight / weightsMax * 100)}% of the largest weight`"
							>
								<div
									class="weight-proportion-value"
									:style="{ width: (weight.weight / weightsMax * 100) + '%' }"
								/>
							</td>
						</tr>
					</template>
				</tbody>
			</table>
		</Card>
	</Modal>
</template>

<script setup lang="ts">
import {
	computed,
	ref,
	shallowReactive,
	watchEffect,
} from 'vue'
import {useI18n} from 'vue-i18n'

import BaseButton from '@/components/base/BaseButton.vue'
import Dropdown from '@/components/misc/Dropdown.vue'
import DropdownItem from '@/components/misc/DropdownItem.vue'
import FancyCheckbox from '@/components/input/FancyCheckbox.vue'
import FilterInput from '@/components/input/filter/FilterInput.vue'
import Filters from '@/components/project/partials/Filters.vue'
import SavedFilterUrgencyWeightsService from '@/services/urgencyWeights'
import TaskFilterParams, { getDefaultTaskFilterParams } from '@/services/taskCollection'
import type {IProject} from '@/modelTypes/IProject'
import {success} from '@/message'
import {useTitle} from '@/composables/useTitle'

const props = defineProps<{
	projectId: IProject['id'],
}>()

const service = shallowReactive(new SavedFilterUrgencyWeightsService())

const {t} = useI18n({useScope: 'global'})

useTitle(() => `${t('project.urgentTasks.title')} - ${t('projects.edit.title')}`)

const allPropertyNames = new Set([
	'due_date',
	'priority',
	'percent_done',
	'matches_filter',
])
const newProperty = ref<string>(null)

const weights = ref<ISavedFilterUrgencyWeight[]>([])
const weightsTotal = ref<number>(0)
const weightsMax = ref<number>(0)
service.get({id: props.projectId}).then((result: ISavedFilterUrgencyWeights) => {
	setWeights(result.urgencyWeights)
	watchEffect(() => {
		weightsTotal.value = weights.value
			.map(w => w.weight)
			.reduce((sum, weight) => sum + weight, 0)
		weightsMax.value = weights.value
			.map(w => w.weight)
			.reduce((max, weight) => weight > max ? weight : max, 0)
		newProperty.value = null
		newPropertyNames()
			.values()
			.take(1)
			.forEach(value => {
				newProperty.value = value
			})
	})
})

const filterEditorIndex = ref<number>(null)
const filterEditorWeight = ref<TaskFilterParams>(null)

function newPropertyNames() {
	return allPropertyNames
		.difference(new Set(weights.value.map(w => w.property)))
		.add('matches_filter') // matches_filter can be added multiple times
}

function setWeights(newWeights: ISavedFilterUrgencyWeight[]) {
	weights.value = newWeights
		.map(w => {
			return {
				...w,
				propertyName() {
					return t('project.urgentTasks.properties.' + this.property)
				},
				displayName() {
					if (this.property === 'matches_filter') {
						return `Matches "${this.filter.query}"`
					}
					return this.propertyName()
				}
			}
		})
}

async function addWeight() {
	const urgencyWeight: ISavedFilterUrgencyWeight = {
		property: newProperty.value,
		weight: 1,
		filter: newProperty.value !== 'matches_filter' ? null : {
			query: 'done = false',
			includeNulls: false,
		},
	}
	await updateWeights(
		weights.value.concat([
			urgencyWeight,
		]))
}

// updateWeight updates the given index with a shallow merge of weight onto the current value
async function updateWeight(index: number, weight: ISavedFilterUrgencyWeight) {
	weight = Object.assign({}, weights.value[index], weight)
	await updateWeights(weights.value.with(index, weight))
}

async function deleteWeight(index: number) {
	await updateWeights(weights.value.toSpliced(index, 1))
}

async function updateWeights(urgencyWeights: ISavedFilterUrgencyWeight[]) {
	urgencyWeights = urgencyWeights.map(w => {
		if (w.property !== 'matches_filter') {
			w.filter = null
		}
		return w
	})
	const response = await service.create({ id: props.projectId, urgencyWeights })
	setWeights(urgencyWeights)
	success(response)
}

function editWeightFilter(index: number) {
	const weight = weights.value[index]
	filterEditorWeight.value = Object.assign(getDefaultTaskFilterParams(), {
		filter: weight.filter.query,
		filter_include_nulls: weight.filter.includeNulls,
		// filter_timezone: weight.filter.timeZone, // TODO
	})
	filterEditorIndex.value = index
}

async function updateWeightFilter(index: number, filter: TaskFilterParams) {
	await updateWeight(index, {
		filter: {
			query: filter.filter,
			includeNulls: filter.filter_include_nulls,
			// timeZone: filter.filter_time_zone, // TODO
		},
	})
	filterEditorIndex.value = null
}
</script>

<style lang="scss" scoped>
.weight-property {
	white-space: nowrap;
}
.weight-filter {
	min-width: 100%;

	.filter-input {
		flex-grow: 1;
		width: 100%;
		height: 100%;
		padding: 0;

		// This particular input is always disabled. Visually indicate it too.
		background-color: var(--input-disabled-background-color);
		&, &:hover {
			border-color: var(--input-disabled-border-color);
		}
	}
}
.weight-input {
	width: 5em;
	flex-grow: 0;
}

.weight-creator {
	align-items: center;
	display: flex;
	flex-direction: row;
	justify-content: start;
	margin-bottom: 1.5em;

	> * {
		margin: 0 0.5em;
	}
}

.property-picker {
	align-items: flex-start;
	display: flex;
	flex-direction: column;
	white-space: nowrap;
}

.weights {
	tr {
		th, td {
			vertical-align: middle;
			border-bottom: none;
			border-top: 2px solid var(--border);
			padding-left: 0.3em; // Increase information density slightly
			padding-right: 0.3em;
		}
		&:not(:first-child) td {
			padding-top: 0.9em;
		}
	}

	.weight-proportion {
		background-color: color-mix(var(--border) 30%);

		.weight-proportion-container {
			border: none;
			padding: 0;
		}
		.weight-proportion-value {
			height: 0.25rem;
			transition: width 0.5s;
		}
		@for $i from 1 through 5 {
			/* Color would normally be 5n + i, but need to only apply to every second row. */
			&:nth-child(10n + #{$i * 2}) .weight-proportion-value {
				background-color: var(--chart-color-#{$i});
			}
		}
	}
}

.actions {
	display: flex;
	align-items: center;
	justify-content: flex-end;
	> :not(:last-child) {
		margin-right: 0.5rem;
	}
}

.urgent-tasks {
	text-align: left;
}
</style>
