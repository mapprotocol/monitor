// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package ethclient

//// mapSenderFromServer is a types.Signer that remembers the sender address returned by the RPC
//// server. It is stored in the transaction's sender address cache to avoid an additional
//// request in TransactionSender.
//type mapSenderFromServer struct {
//	addr      common.Address
//	blockhash common.Hash
//}
//
//func setMAPSenderFromServer(tx *types.Transaction, addr common.Address, block common.Hash) {
//	// Use types.Sender for side-effect to store our signer into the cache.
//	types.Sender(&mapSenderFromServer{addr, block}, tx)
//}
//
//func (s *mapSenderFromServer) Equal(other types.Signer) bool {
//	os, ok := other.(*mapSenderFromServer)
//	return ok && os.blockhash == s.blockhash
//}
//
//func (s *mapSenderFromServer) Sender(tx *types.Transaction) (common.Address, error) {
//	if s.blockhash == (common.Hash{}) {
//		return common.Address{}, errNotCached
//	}
//	return s.addr, nil
//}
//
//func (s *mapSenderFromServer) ChainID() *big.Int {
//	panic("can't sign with senderFromServer")
//}
//func (s *mapSenderFromServer) Hash(tx *types.Transaction) common.Hash {
//	panic("can't sign with senderFromServer")
//}
//func (s *mapSenderFromServer) SignatureValues(tx *types.Transaction, sig []byte) (R, S, V *big.Int, err error) {
//	panic("can't sign with senderFromServer")
//}
