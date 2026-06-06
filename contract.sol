// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/**
 * @title SimpleStorage
 * @dev This is the Solidity equivalent of the raw EVM bytecode currently executed
 * automatically by the node's transaction generator in dev mode.
 *
 * What the raw bytecode does:
 * 1. PUSH1 5, PUSH1 0, SSTORE   => storedData = 5;
 * 2. PUSH1 0, SLOAD             => retrieves 5 into the EVM stack
 * 3. PUSH1 0, MSTORE            => stores 5 into EVM memory
 * 4. PUSH1 0x20, PUSH1 0, RETURN=> returns the 32 bytes from memory
 */
contract SimpleStorage {
    uint256 public storedData;

    function storeAndReturn() public returns (uint256) {
        // Store the value 5 in slot 0
        storedData = 5;

        // Return the value (which gets returned by the block)
        return storedData;
    }
}
