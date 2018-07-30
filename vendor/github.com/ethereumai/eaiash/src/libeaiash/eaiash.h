/*
  This file is part of eaiash.

  eaiash is free software: you can redistribute it and/or modify
  it under the terms of the GNU General Public License as published by
  the Free Software Foundation, either version 3 of the License, or
  (at your option) any later version.

  eaiash is distributed in the hope that it will be useful,
  but WITHOUT ANY WARRANTY; without even the implied warranty of
  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
  GNU General Public License for more details.

  You should have received a copy of the GNU General Public License
  along with eaiash.  If not, see <http://www.gnu.org/licenses/>.
*/

/** @file eaiash.h
* @date 2015
*/
#pragma once

#include <stdint.h>
#include <stdbool.h>
#include <string.h>
#include <stddef.h>
#include "compiler.h"

#define EAIASH_REVISION 23
#define EAIASH_DATASET_BYTES_INIT 1073741824U // 2**30
#define EAIASH_DATASET_BYTES_GROWTH 8388608U  // 2**23
#define EAIASH_CACHE_BYTES_INIT 1073741824U // 2**24
#define EAIASH_CACHE_BYTES_GROWTH 131072U  // 2**17
#define EAIASH_EPOCH_LENGTH 30000U
#define EAIASH_MIX_BYTES 128
#define EAIASH_HASH_BYTES 64
#define EAIASH_DATASET_PARENTS 256
#define EAIASH_CACHE_ROUNDS 3
#define EAIASH_ACCESSES 64
#define EAIASH_DAG_MAGIC_NUM_SIZE 8
#define EAIASH_DAG_MAGIC_NUM 0xFEE1DEADBADDCAFE

#ifdef __cplusplus
extern "C" {
#endif

/// Type of a seedhash/blockhash e.t.c.
typedef struct eaiash_h256 { uint8_t b[32]; } eaiash_h256_t;

// convenience macro to statically initialize an h256_t
// usage:
// eaiash_h256_t a = eaiash_h256_static_init(1, 2, 3, ... )
// have to provide all 32 values. If you don't provide all the rest
// will simply be unitialized (not guranteed to be 0)
#define eaiash_h256_static_init(...)			\
	{ {__VA_ARGS__} }

struct eaiash_light;
typedef struct eaiash_light* eaiash_light_t;
struct eaiash_full;
typedef struct eaiash_full* eaiash_full_t;
typedef int(*eaiash_callback_t)(unsigned);

typedef struct eaiash_return_value {
	eaiash_h256_t result;
	eaiash_h256_t mix_hash;
	bool success;
} eaiash_return_value_t;

/**
 * Allocate and initialize a new eaiash_light handler
 *
 * @param block_number   The block number for which to create the handler
 * @return               Newly allocated eaiash_light handler or NULL in case of
 *                       ERRNOMEM or invalid parameters used for @ref eaiash_compute_cache_nodes()
 */
eaiash_light_t eaiash_light_new(uint64_t block_number);
/**
 * Frees a previously allocated eaiash_light handler
 * @param light        The light handler to free
 */
void eaiash_light_delete(eaiash_light_t light);
/**
 * Calculate the light client data
 *
 * @param light          The light client handler
 * @param header_hash    The header hash to pack into the mix
 * @param nonce          The nonce to pack into the mix
 * @return               an object of eaiash_return_value_t holding the return values
 */
eaiash_return_value_t eaiash_light_compute(
	eaiash_light_t light,
	eaiash_h256_t const header_hash,
	uint64_t nonce
);

/**
 * Allocate and initialize a new eaiash_full handler
 *
 * @param light         The light handler containing the cache.
 * @param callback      A callback function with signature of @ref eaiash_callback_t
 *                      It accepts an unsigned with which a progress of DAG calculation
 *                      can be displayed. If all goes well the callback should return 0.
 *                      If a non-zero value is returned then DAG generation will stop.
 *                      Be advised. A progress value of 100 means that DAG creation is
 *                      almost complete and that this function will soon return succesfully.
 *                      It does not mean that the function has already had a succesfull return.
 * @return              Newly allocated eaiash_full handler or NULL in case of
 *                      ERRNOMEM or invalid parameters used for @ref eaiash_compute_full_data()
 */
eaiash_full_t eaiash_full_new(eaiash_light_t light, eaiash_callback_t callback);

/**
 * Frees a previously allocated eaiash_full handler
 * @param full    The light handler to free
 */
void eaiash_full_delete(eaiash_full_t full);
/**
 * Calculate the full client data
 *
 * @param full           The full client handler
 * @param header_hash    The header hash to pack into the mix
 * @param nonce          The nonce to pack into the mix
 * @return               An object of eaiash_return_value to hold the return value
 */
eaiash_return_value_t eaiash_full_compute(
	eaiash_full_t full,
	eaiash_h256_t const header_hash,
	uint64_t nonce
);
/**
 * Get a pointer to the full DAG data
 */
void const* eaiash_full_dag(eaiash_full_t full);
/**
 * Get the size of the DAG data
 */
uint64_t eaiash_full_dag_size(eaiash_full_t full);

/**
 * Calculate the seedhash for a given block number
 */
eaiash_h256_t eaiash_get_seedhash(uint64_t block_number);

#ifdef __cplusplus
}
#endif
