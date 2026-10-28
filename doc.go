// Package randomizer generates unbiased random numbers, byte strings, network
// data, telecom identifiers, permutations, and samples.
// Package functions and [New] use a fast, concurrently safe source.
// [NewSeeded] supports reproducible sequential calls, [NewSecure] supplies
// cryptographic randomness for secrets, and [FromSource] and [FromReader]
// bind custom inputs to a generator.
// Random results can repeat; randomness does not imply uniqueness.
// Callers own output buffers and must synchronize overlapping writes.
package randomizer
