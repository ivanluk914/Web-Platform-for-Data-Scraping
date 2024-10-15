import { useState, useMemo, useCallback } from "react";
import {
  Table,
  TableHeader,
  TableColumn,
  TableBody,
  TableRow,
  TableCell,
  User,
  Chip,
  Pagination,
  Input,
  Button,
  DropdownTrigger,
  Dropdown,
  DropdownMenu,
  DropdownItem,
  SharedSelection,
} from "@nextui-org/react";
import { BsSearch, BsChevronDown } from "react-icons/bs";
import { User as UserModel } from "../models/user";

const columns = [
  { name: "NAME", uid: "name" },
  { name: "EMAIL", uid: "email" },
  { name: "NICKNAME", uid: "nickname" },
  { name: "LAST LOGIN", uid: "last_login" },
];

interface UserTableProps {
  users: UserModel[];
  isLoading: boolean;
  error: Error | null;
  page: number;
  pageSize: number;
  setPage: (page: number) => void;
  setPageSize: (pageSize: number) => void;
}

const UserTable = ({ 
  users, 
  isLoading, 
  error, 
  page, 
  pageSize, 
  setPage, 
  setPageSize 
}: UserTableProps) => {
  const [filterValue, setFilterValue] = useState("");
  const [visibleColumns, setVisibleColumns] = useState(new Set(columns.map(col => col.uid)));

  console.log(111, users)

  const filteredItems = useMemo(() => {
    return users.filter((user) =>
      user.name.toLowerCase().includes(filterValue.toLowerCase()) ||
      user.email.toLowerCase().includes(filterValue.toLowerCase()) ||
      user.nickname.toLowerCase().includes(filterValue.toLowerCase())
    );
  }, [users, filterValue]);

  const pages = Math.ceil(filteredItems.length / pageSize);

  const renderCell = useCallback((user: UserModel, columnKey: React.Key) => {
    const cellValue = user[columnKey as keyof UserModel];

    switch (columnKey) {
      case "name":
        return (
          <User
            avatarProps={{ radius: "lg", src: user.picture }}
            name={cellValue}
          >
            {user.email}
          </User>
        );
      case "last_login":
        return (
          <Chip className="capitalize" size="sm" variant="flat">
            {new Date(cellValue as string).toLocaleString()}
          </Chip>
        );
      default:
        return cellValue;
    }
  }, []);

  const topContent = useMemo(() => {
    return (
      <div className="flex flex-col gap-4">
        <div className="flex justify-between gap-3 items-end">
          <Input
            isClearable
            className="w-full sm:max-w-[44%]"
            placeholder="Search by name, email, or nickname..."
            startContent={<BsSearch />}
            value={filterValue}
            onClear={() => setFilterValue("")}
            onValueChange={setFilterValue}
          />
          <Dropdown>
            <DropdownTrigger className="hidden sm:flex">
              <Button
                endContent={<BsChevronDown className="text-small" />}
                variant="flat"
              >
                Columns
              </Button>
            </DropdownTrigger>
            <DropdownMenu
              disallowEmptySelection
              aria-label="Table Columns"
              closeOnSelect={false}
              selectedKeys={visibleColumns}
              selectionMode="multiple"
              onSelectionChange={setVisibleColumns}
            >
              {columns.map((column) => (
                <DropdownItem key={column.uid} className="capitalize">
                  {column.name}
                </DropdownItem>
              ))}
            </DropdownMenu>
          </Dropdown>
        </div>
      </div>
    );
  }, [filterValue, visibleColumns]);

  const bottomContent = useMemo(() => {
    return (
      <div className="py-2 px-2 flex justify-between items-center">
        <Pagination
          isCompact
          showControls
          showShadow
          color="primary"
          page={page}
          total={pages}
          onChange={setPage}
        />
        <div className="hidden sm:flex w-[30%] justify-end gap-2">
          <Button
            isDisabled={page === 1}
            size="sm"
            variant="flat"
            onPress={() => setPage(page - 1)}
          >
            Previous
          </Button>
          <Button
            isDisabled={page === pages}
            size="sm"
            variant="flat"
            onPress={() => setPage(page + 1)}
          >
            Next
          </Button>
        </div>
      </div>
    );
  }, [page, pages, setPage]);

  if (isLoading) {
    return <div>Loading...</div>;
  }

  if (error) {
    return <div>Error: {error.message}</div>;
  }

  return (
    <Table
      aria-label="User table"
      isHeaderSticky
      bottomContent={bottomContent}
      bottomContentPlacement="outside"
      classNames={{
        wrapper: "max-h-[382px]",
      }}
      topContent={topContent}
      topContentPlacement="outside"
    >
      <TableHeader columns={columns.filter(col => visibleColumns.has(col.uid))}>
        {(column) => (
          <TableColumn key={column.uid}>
            {column.name}
          </TableColumn>
        )}
      </TableHeader>
      <TableBody items={filteredItems.slice((page - 1) * pageSize, page * pageSize)}>
        {(item) => (
          <TableRow key={item.user_id}>
            {(columnKey) => (
              <TableCell>{renderCell(item, columnKey)}</TableCell>
            )}
          </TableRow>
        )}
      </TableBody>
    </Table>
  );
};

export default UserTable;